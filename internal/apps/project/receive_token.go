/*
 * MIT License
 *
 * Copyright (c) 2025 linux.do
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package project

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linux-do/cdk/internal/apps/oauth"
	"github.com/linux-do/cdk/internal/db"
	"github.com/redis/go-redis/v9"
)

// 领取凭证（receive token）
//
// 领取分两步：
//  1. GET  /projects/:id/receive/token  服务端签发一次性凭证（仅项目可领取时签发，绑定 用户+项目）
//  2. POST /projects/:id/receive        提交 captcha_token + receive_token
//
// 凭证只能在项目开始后取得、只能用一次、有有效期，
// 因此提前过验证码囤 token、重放旧请求都无法直接领取。

const (
	// receiveTokenTTL 凭证有效期
	receiveTokenTTL = 2 * time.Minute
	// receiveTokenBytes 凭证随机字节数
	receiveTokenBytes = 32
	// receiveBodyMaxBytes 领取请求体上限
	receiveBodyMaxBytes = 64 << 10
)

// receiveTokenConsumeScript 原子比较并删除：值一致才删除并返回 1，保证一次性
var receiveTokenConsumeScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
end
return 0
`)

// ReceiveTokenResponseData 领取凭证响应
type ReceiveTokenResponseData struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

type receiveTokenBody struct {
	ReceiveToken string `json:"receive_token"`
}

func receiveTokenKey(projectID string, userID uint64) string {
	return fmt.Sprintf("project:%s:receive_token:%d", projectID, userID)
}

// GetReceiveToken
// @Tags project
// @Summary 获取领取凭证
// @Description 项目可领取时签发一次性凭证，领取时需与 captcha_token 一并提交
// @Produce json
// @Param id path string true "项目ID"
// @Success 200 {object} ProjectResponse{data=ReceiveTokenResponseData}
// @Router /api/v1/projects/{id}/receive/token [get]
func GetReceiveToken(c *gin.Context) {
	ctx := c.Request.Context()
	project, ok := GetProjectFromContext(c)
	if !ok || project == nil {
		c.JSON(http.StatusInternalServerError, ProjectResponse{ErrorMsg: UnknownError})
		return
	}

	raw := make([]byte, receiveTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		c.JSON(http.StatusInternalServerError, ProjectResponse{ErrorMsg: err.Error()})
		return
	}
	token := hex.EncodeToString(raw)

	if err := db.Redis.Set(ctx, receiveTokenKey(project.ID, oauth.GetUserIDFromContext(c)), token, receiveTokenTTL).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, ProjectResponse{ErrorMsg: err.Error()})
		return
	}

	c.JSON(http.StatusOK, ProjectResponse{Data: ReceiveTokenResponseData{
		Token:     token,
		ExpiresIn: int(receiveTokenTTL / time.Second),
	}})
}

// ReceiveTokenMiddleware 校验并消费领取凭证；需在 ReceiveProjectMiddleware 之后
func ReceiveTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		project, ok := GetProjectFromContext(c)
		if !ok || project == nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, ProjectResponse{ErrorMsg: UnknownError})
			return
		}

		// 读取 body 并回填，保证后续 handler 仍可读取
		var body receiveTokenBody
		if c.Request.Body != nil {
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, receiveBodyMaxBytes))
			_ = c.Request.Body.Close()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, ProjectResponse{ErrorMsg: ReceiveTokenInvalid})
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(raw))
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &body)
			}
		}

		if body.ReceiveToken == "" || len(body.ReceiveToken) != receiveTokenBytes*2 {
			c.AbortWithStatusJSON(http.StatusBadRequest, ProjectResponse{ErrorMsg: ReceiveTokenInvalid})
			return
		}

		key := receiveTokenKey(project.ID, oauth.GetUserIDFromContext(c))
		n, err := receiveTokenConsumeScript.Run(ctx, db.Redis, []string{key}, body.ReceiveToken).Int()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, ProjectResponse{ErrorMsg: err.Error()})
			return
		}
		if n != 1 {
			c.AbortWithStatusJSON(http.StatusBadRequest, ProjectResponse{ErrorMsg: ReceiveTokenInvalid})
			return
		}

		c.Next()
	}
}

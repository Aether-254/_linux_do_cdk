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

package payment

import (
	"errors"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMarkOrderPaidReturnsUpdateError(t *testing.T) {
	dialector := mysql.New(mysql.Config{
		DSN:                       "unused:unused@tcp(127.0.0.1:1)/unused",
		SkipInitializeWithVersion: true,
	})
	database, err := gorm.Open(dialector, &gorm.Config{
		DryRun:                 true,
		DisableAutomaticPing:   true,
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("open dry-run database: %v", err)
	}

	wantErr := errors.New("database write failed")
	if err := database.Callback().Update().Before("gorm:update").Register("test:fail_order_update", func(tx *gorm.DB) {
		tx.AddError(wantErr)
	}); err != nil {
		t.Fatalf("register update callback: %v", err)
	}

	updated, err := markOrderPaid(database, "order-1", "trade-1", time.Now())
	if !errors.Is(err, wantErr) {
		t.Fatalf("markOrderPaid error = %v, want %v", err, wantErr)
	}
	if updated {
		t.Fatal("markOrderPaid reported an update after the database rejected it")
	}
}

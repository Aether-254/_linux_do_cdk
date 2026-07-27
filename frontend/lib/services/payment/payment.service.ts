import apiClient from '../core/api-client';
import {
  GetPaymentConfigResponse,
  PaymentConfigData,
  UpsertPaymentConfigRequest,
  UpsertPaymentConfigResponse,
  GetPendingPaymentResponse,
  PendingPaymentData,
} from './types';

/**
 * 支付相关 API 服务
 */
export class PaymentService {
  private static readonly base = '/api/v1/users/payment-config';

  static async getPendingPayment(projectId: string): Promise<PendingPaymentData> {
    const response = await apiClient.get<GetPendingPaymentResponse>(`/api/v1/projects/${projectId}/pending-payment`);
    if (response.data.error_msg) {
      throw new Error(response.data.error_msg);
    }
    return response.data.data;
  }

  static async getPendingPaymentSafe(projectId: string): Promise<{
    success: boolean;
    data?: PendingPaymentData;
    error?: string;
  }> {
    try {
      const data = await this.getPendingPayment(projectId);
      return {success: true, data};
    } catch (err) {
      return {success: false, error: err instanceof Error ? err.message : '获取待支付订单失败'};
    }
  }

  static async getConfig(): Promise<PaymentConfigData> {
    const response = await apiClient.get<GetPaymentConfigResponse>(this.base);
    if (response.data.error_msg) {
      throw new Error(response.data.error_msg);
    }
    return response.data.data;
  }

  static async getConfigSafe(): Promise<{success: boolean; data?: PaymentConfigData; error?: string}> {
    try {
      const data = await this.getConfig();
      return {success: true, data};
    } catch (err) {
      return {success: false, error: err instanceof Error ? err.message : '获取支付配置失败'};
    }
  }

  static async updateConfig(payload: UpsertPaymentConfigRequest): Promise<void> {
    const response = await apiClient.put<UpsertPaymentConfigResponse>(this.base, payload);
    if (response.data.error_msg) {
      throw new Error(response.data.error_msg);
    }
  }

  static async updateConfigSafe(payload: UpsertPaymentConfigRequest): Promise<{success: boolean; error?: string}> {
    try {
      await this.updateConfig(payload);
      return {success: true};
    } catch (err) {
      return {success: false, error: err instanceof Error ? err.message : '保存支付配置失败'};
    }
  }

  static async deleteConfig(): Promise<void> {
    const response = await apiClient.delete<UpsertPaymentConfigResponse>(this.base);
    if (response.data.error_msg) {
      throw new Error(response.data.error_msg);
    }
  }

  static async deleteConfigSafe(): Promise<{success: boolean; error?: string}> {
    try {
      await this.deleteConfig();
      return {success: true};
    } catch (err) {
      return {success: false, error: err instanceof Error ? err.message : '删除支付配置失败'};
    }
  }
}

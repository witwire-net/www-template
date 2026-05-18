/**
 * サービス層で使用する共通エラークラス。
 * HTTP ステータスコードと機械可読なエラーコードを持ち、
 * 呼び出し側（route / handler）で適切なレスポンスに変換することを想定する。
 */
export class ServiceError extends Error {
  /**
   * @param message 人間可読なエラーメッセージ
   * @param statusCode HTTP ステータスコード相当の数値
   * @param code 機械可読なエラーコード識別子
   */
  constructor(
    message: string,
    public readonly statusCode: number,
    public readonly code: string
  ) {
    super(message);
    this.name = 'ServiceError';
    Object.setPrototypeOf(this, ServiceError.prototype);
  }
}

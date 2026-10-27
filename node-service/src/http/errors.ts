// Error yang membawa status HTTP. Padanan AppError (Rust) / HttpError (PHP).
export class HttpError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }

  static badRequest(message: string): HttpError {
    return new HttpError(400, message);
  }

  static notFound(message: string): HttpError {
    return new HttpError(404, message);
  }
}

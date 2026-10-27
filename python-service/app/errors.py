"""Exception yang membawa status HTTP. Padanan AppError (Rust) / HttpError (PHP, Node)."""


class ApiError(Exception):
    def __init__(self, status: int, message: str) -> None:
        super().__init__(message)
        self.status = status
        self.message = message

    @classmethod
    def bad_request(cls, message: str) -> "ApiError":
        return cls(400, message)

    @classmethod
    def not_found(cls, message: str) -> "ApiError":
        return cls(404, message)

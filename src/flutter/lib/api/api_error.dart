class ApiError implements Exception {
  ApiError.http(this.status, this.message) : type = ApiErrorType.http;

  ApiError.transport(this.message)
      : status = null,
        type = ApiErrorType.transport;

  ApiError.decode(this.message)
      : status = null,
        type = ApiErrorType.decode;

  final ApiErrorType type;
  final int? status;
  final String message;

  @override
  String toString() => message;
}

enum ApiErrorType { http, transport, decode }

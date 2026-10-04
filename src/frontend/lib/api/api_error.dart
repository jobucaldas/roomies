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

/// User-facing text for [error]. Network failures and 5xx responses become
/// [unreachable] instead of a bare status code or socket message.
String describeError(Object error, {required String unreachable}) {
  if (error is ApiError) {
    final status = error.status;
    if (error.type == ApiErrorType.transport ||
        (status != null && status >= 500)) {
      return unreachable;
    }
    return error.message;
  }
  return error.toString();
}

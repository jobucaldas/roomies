bool isWorkOSAuthorizeUrl(String url) {
  final uri = Uri.tryParse(url);
  if (uri == null) return false;
  return uri.scheme == 'https' &&
      uri.host == 'api.workos.com' &&
      uri.path == '/user_management/authorize';
}

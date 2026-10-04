bool isWorkOSAuthorizeUrl(String url) =>
    _isWorkOSPath(url, '/user_management/authorize');

/// Hosted AuthKit logout (ends the WorkOS session, then redirects to the
/// dashboard's configured logout URL).
bool isWorkOSLogoutUrl(String url) =>
    _isWorkOSPath(url, '/user_management/sessions/logout');

bool _isWorkOSPath(String url, String path) {
  final uri = Uri.tryParse(url);
  if (uri == null) return false;
  return uri.scheme == 'https' &&
      uri.host == 'api.workos.com' &&
      uri.path == path;
}

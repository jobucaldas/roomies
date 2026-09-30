import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';

class PkcePair {
  const PkcePair({
    required this.verifier,
    required this.challenge,
    this.challengeMethod = 'S256',
  });

  final String verifier;
  final String challenge;
  final String challengeMethod;
}

PkcePair generatePkcePair() {
  final random = Random.secure();
  final bytes = Uint8List.fromList(
    List<int>.generate(32, (_) => random.nextInt(256)),
  );
  final verifier = base64UrlEncode(bytes).replaceAll('=', '');
  final digest = sha256.convert(utf8.encode(verifier));
  final challenge = base64UrlEncode(digest.bytes).replaceAll('=', '');
  return PkcePair(verifier: verifier, challenge: challenge);
}

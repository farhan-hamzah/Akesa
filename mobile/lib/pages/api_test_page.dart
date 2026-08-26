import 'dart:convert';

import 'package:clerk_flutter/clerk_flutter.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

class ApiTestPage extends StatefulWidget {
  final ClerkAuthState authState;

  const ApiTestPage({super.key, required this.authState});

  @override
  State<ApiTestPage> createState() => _ApiTestPageState();
}

class _ApiTestPageState extends State<ApiTestPage> {
  String result = 'Belum ada request';
  bool loading = false;

  static const String baseUrl = 'http://10.0.2.2:9090';

  // ============================================================
  // GET CLERK SESSION TOKEN
  // ============================================================

  Future<String?> getSessionToken() async {
    try {
      if (!widget.authState.isSignedIn) {
        if (mounted) {
          setState(() {
            result = 'User belum login ke Clerk.';
          });
        }

        return null;
      }

      final sessionToken = await widget.authState.sessionToken();

      return sessionToken.jwt;
    } catch (e, stackTrace) {
      debugPrint('Gagal mengambil Clerk token: $e');
      debugPrint('$stackTrace');

      if (mounted) {
        setState(() {
          result = 'Gagal mendapatkan Clerk session token:\n$e';
        });
      }

      return null;
    }
  }

  // ============================================================
  // HEALTH CHECK
  // ============================================================

  Future<void> testHealth() async {
    setState(() {
      loading = true;
      result = 'Menghubungi backend...';
    });

    try {
      final response = await http.get(Uri.parse('$baseUrl/health'));

      setState(() {
        result =
            '''
STATUS: ${response.statusCode}

${response.body}
''';
      });
    } catch (e) {
      setState(() {
        result = 'ERROR:\n$e';
      });
    } finally {
      if (mounted) {
        setState(() {
          loading = false;
        });
      }
    }
  }

  // ============================================================
  // GET /api/v1/me
  // ============================================================

  Future<void> testMe() async {
    setState(() {
      loading = true;
      result = 'Mendapatkan Clerk session token...';
    });

    try {
      final token = await getSessionToken();

      if (token == null || token.isEmpty) {
        setState(() {
          result = 'Session token tidak tersedia.';
        });
        return;
      }

      debugPrint('Mengirim token ke /api/v1/me');

      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/me'),
        headers: {
          'Authorization': 'Bearer $token',
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
      );

      String body = response.body;

      try {
        final json = jsonDecode(response.body);

        body = const JsonEncoder.withIndent('  ').convert(json);
      } catch (_) {
        // Response bukan JSON
      }

      setState(() {
        result =
            '''
STATUS: ${response.statusCode}

$body
''';
      });
    } catch (e) {
      setState(() {
        result = 'ERROR:\n$e';
      });
    } finally {
      if (mounted) {
        setState(() {
          loading = false;
        });
      }
    }
  }

  // ============================================================
  // POST /api/v1/users/sync
  // ============================================================

  Future<void> testSync() async {
    setState(() {
      loading = true;
      result = 'Mendapatkan Clerk session token...';
    });

    try {
      final token = await getSessionToken();

      if (token == null || token.isEmpty) {
        setState(() {
          result = 'Session token tidak tersedia.';
        });
        return;
      }

      final response = await http.post(
        Uri.parse('$baseUrl/api/v1/users/sync'),
        headers: {
          'Authorization': 'Bearer $token',
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
      );

      String body = response.body;

      try {
        final json = jsonDecode(response.body);

        body = const JsonEncoder.withIndent('  ').convert(json);
      } catch (_) {}

      setState(() {
        result =
            '''
STATUS: ${response.statusCode}

$body
''';
      });
    } catch (e) {
      setState(() {
        result = 'ERROR:\n$e';
      });
    } finally {
      if (mounted) {
        setState(() {
          loading = false;
        });
      }
    }
  }

  // ============================================================
  // SHOW SESSION TOKEN
  // ============================================================

  Future<void> showSessionToken() async {
    setState(() {
      loading = true;
      result = 'Mendapatkan session token...';
    });

    try {
      final token = await getSessionToken();

      if (token == null || token.isEmpty) {
        setState(() {
          result = 'Session token tidak tersedia.';
        });
        return;
      }

      setState(() {
        result =
            '''
CLERK SESSION TOKEN:

$token
''';
      });
    } catch (e) {
      setState(() {
        result = 'ERROR:\n$e';
      });
    } finally {
      if (mounted) {
        setState(() {
          loading = false;
        });
      }
    }
  }

  // ============================================================
  // LOGOUT
  // ============================================================

  Future<void> logout() async {
    try {
      setState(() {
        loading = true;
        result = 'Logging out...';
      });

      await widget.authState.signOut();

      debugPrint('Clerk logout berhasil.');
    } catch (e, stackTrace) {
      debugPrint('Gagal logout: $e');
      debugPrint('$stackTrace');

      if (mounted) {
        setState(() {
          result = 'Gagal logout:\n$e';
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          loading = false;
        });
      }
    }
  }

  // ============================================================
  // UI
  // ============================================================

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Akesa API Test'),
        actions: [
          IconButton(
            tooltip: 'Logout',
            onPressed: loading ? null : logout,
            icon: const Icon(Icons.logout),
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'Akesa Backend Test',
              style: TextStyle(fontSize: 24, fontWeight: FontWeight.bold),
            ),

            const SizedBox(height: 8),

            const Text('Flutter → Clerk → Go API → PostgreSQL'),

            const SizedBox(height: 24),

            FilledButton(
              onPressed: loading ? null : testHealth,
              child: const Text('GET /health'),
            ),

            const SizedBox(height: 8),

            FilledButton(
              onPressed: loading ? null : showSessionToken,
              child: const Text('Get Clerk Session Token'),
            ),

            const SizedBox(height: 8),

            FilledButton(
              onPressed: loading ? null : testMe,
              child: const Text('GET /api/v1/me'),
            ),

            const SizedBox(height: 8),

            FilledButton(
              onPressed: loading ? null : testSync,
              child: const Text('POST /api/v1/users/sync'),
            ),

            const SizedBox(height: 24),

            const Text(
              'Response',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
            ),

            const SizedBox(height: 8),

            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                border: Border.all(color: Colors.grey),
                borderRadius: BorderRadius.circular(8),
              ),
              child: SelectableText(
                result,
                style: const TextStyle(fontFamily: 'monospace'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

import 'dart:convert';

import 'package:clerk_flutter/clerk_flutter.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

class ApiTestPage extends StatefulWidget {
  const ApiTestPage({super.key});

  @override
  State<ApiTestPage> createState() => _ApiTestPageState();
}

class _ApiTestPageState extends State<ApiTestPage> {
  String result = 'Belum ada request';

  bool loading = false;

  static const String baseUrl = 'http://10.0.2.2:9090';

  Future<void> testHealth() async {
    setState(() {
      loading = true;
      result = 'Loading...';
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
      setState(() {
        loading = false;
      });
    }
  }

  Future<String?> getSessionToken() async {
    setState(() {
      result =
          'Session token belum dapat diambil.\n'
          'API session Clerk perlu disesuaikan dengan versi clerk_flutter.';
    });

    return null;
  }

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

      setState(() {
        result = '''
Session token berhasil didapatkan.

Mengirim request ke Go API...
''';
      });

      final response = await http.get(
        Uri.parse('$baseUrl/api/v1/me'),
        headers: {
          'Authorization': 'Bearer $token',
          'Content-Type': 'application/json',
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
      setState(() {
        loading = false;
      });
    }
  }

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
      setState(() {
        loading = false;
      });
    }
  }

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
      setState(() {
        loading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Akesa API Test')),
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

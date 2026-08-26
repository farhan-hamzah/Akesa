import 'package:clerk_flutter/clerk_flutter.dart';
import 'package:flutter/material.dart';

import 'api_test_page.dart';

class AuthPage extends StatelessWidget {
  const AuthPage({super.key});

  @override
  Widget build(BuildContext context) {
    return ClerkErrorListener(
      child: ClerkAuthBuilder(
        signedInBuilder: (context, authState) {
          return ApiTestPage(authState: authState);
        },

        signedOutBuilder: (context, authState) {
          return const Scaffold(body: SafeArea(child: ClerkAuthentication()));
        },
      ),
    );
  }
}

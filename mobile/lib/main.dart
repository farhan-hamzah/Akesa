import 'package:clerk_flutter/clerk_flutter.dart';
import 'package:flutter/material.dart';

import 'config/clerk_config.dart';
import 'pages/auth_page.dart';

void main() {
  runApp(
    ClerkAuth(
      config: ClerkAuthConfig(publishableKey: ClerkConfig.publishableKey),
      child: const AkesaApp(),
    ),
  );
}

class AkesaApp extends StatelessWidget {
  const AkesaApp({super.key});
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Akesa',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
        useMaterial3: true,
      ),
      home: const AuthPage(),
    );
  }
}

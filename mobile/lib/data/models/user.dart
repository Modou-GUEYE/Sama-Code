class User {
  const User({
    required this.id,
    required this.displayName,
    required this.phoneNumber,
    required this.preferredLanguage,
    required this.learningMode,
    required this.roles,
  });

  factory User.fromJson(Map<String, dynamic> json) => User(
    id: json['id'] as String,
    displayName: json['display_name'] as String,
    phoneNumber: json['phone_number'] as String,
    preferredLanguage: json['preferred_language'] as String,
    learningMode: json['learning_mode'] as String,
    roles: (json['roles'] as List).cast<String>(),
  );

  final String id;
  final String displayName;
  final String phoneNumber;
  final String preferredLanguage; // fr | wo
  final String learningMode; // text | audio
  final List<String> roles;
}

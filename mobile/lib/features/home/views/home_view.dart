import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/localization/app_strings.dart';
import '../../../core/session_viewmodel.dart';
import '../../../core/theme/app_theme.dart';
import '../../../data/models/license_category.dart';
import '../../../data/models/exam.dart';
import '../../../data/repositories/exam_repository.dart';
import '../../../data/repositories/license_repository.dart';
import '../../exam/views/exam_view.dart';
import '../../exam/views/series_list_view.dart';
import '../../practice/views/practice_home_view.dart';
import '../../../shared/widgets/feedback.dart';
import '../../../shared/widgets/surfaces.dart';
import '../../notifications/views/notifications_view.dart';
import '../../settings/views/settings_view.dart';
import '../viewmodels/home_viewmodel.dart';

class HomeView extends StatelessWidget {
  const HomeView({super.key});

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (context) => HomeViewModel(
        context.read<LicenseRepository>(),
        context.read<ExamRepository>(),
      )..load(),
      child: const _HomeBody(),
    );
  }
}

class _HomeBody extends StatelessWidget {
  const _HomeBody();

  void _soon(BuildContext context, AppStrings s) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        SnackBar(
          content: Row(
            children: [
              const Icon(Icons.schedule_rounded, color: Colors.white),
              const SizedBox(width: 12),
              Text(s.t('soon'), style: const TextStyle(fontSize: 18)),
            ],
          ),
        ),
      );
  }

  @override
  Widget build(BuildContext context) {
    final session = context.watch<SessionViewModel>();
    final vm = context.watch<HomeViewModel>();
    final s = session.strings;

    return Scaffold(
      body: GradientBackground(
        child: SafeArea(
          child: RefreshIndicator(
            onRefresh: () => Future.wait([vm.load(), session.reloadUser()]),
            child: ListView(
              padding: const EdgeInsets.fromLTRB(20, 16, 20, 32),
              children: [
                _Header(name: session.user?.displayName, strings: s),
                const SizedBox(height: 20),
                if (vm.currentExam != null)
                  _ResumeBanner(
                    exam: vm.currentExam!,
                    strings: s,
                    onTap: () async {
                      await Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => ExamView(session: vm.currentExam!),
                        ),
                      );
                      await vm.load();
                    },
                  )
                else
                  _HeroBanner(strings: s),
                const SizedBox(height: 28),
                _SectionTitle(s.t('learn')),
                Row(
                  children: [
                    Expanded(
                      child: _FeatureTile(
                        icon: Icons.menu_book_rounded,
                        label: s.t('courses'),
                        badge: s.t('soon'),
                        color: AppColors.green,
                        onTap: () => _soon(context, s),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _FeatureTile(
                        icon: Icons.quiz_rounded,
                        label: s.t('practice'),
                        badge: s.t('practice_badge'),
                        color: AppColors.orange,
                        onTap: () async {
                          await Navigator.of(context).push(
                            MaterialPageRoute(
                              builder: (_) => const PracticeHomeView(),
                            ),
                          );
                          await vm.load();
                        },
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 28),
                _SectionTitle(
                  s.t('exam'),
                  subtitle: s.t('exam_choose_license'),
                ),
                if (vm.loading && vm.categories.isEmpty)
                  const Padding(
                    padding: EdgeInsets.all(32),
                    child: Center(child: CircularProgressIndicator()),
                  )
                else if (vm.error != null) ...[
                  ErrorMessage(s.error(vm.error!.code, vm.error!.message)),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    icon: const Icon(Icons.refresh_rounded, size: 28),
                    label: Text(s.t('retry')),
                    onPressed: vm.load,
                  ),
                ] else
                  for (final category in vm.categories) ...[
                    _LicenseCard(
                      category: category,
                      strings: s,
                      onTap: () async {
                        await Navigator.of(context).push(
                          MaterialPageRoute(
                            builder: (_) => SeriesListView(license: category),
                          ),
                        );
                        await vm.load();
                      },
                    ),
                    const SizedBox(height: 16),
                  ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Barre du haut : avatar (paramètres) + salutation à gauche, cloche à droite.
class _Header extends StatelessWidget {
  const _Header({required this.name, required this.strings});

  final String? name;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    return Row(
      children: [
        Semantics(
          button: true,
          label: s.t('settings'),
          excludeSemantics: true,
          child: InkWell(
            customBorder: const CircleBorder(),
            onTap: () => Navigator.of(context)
                .push(MaterialPageRoute(builder: (_) => const SettingsView())),
            child: InitialsAvatar(name: name ?? '', size: 56),
          ),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                s.t('hello'),
                style: const TextStyle(fontSize: 16, color: Ink2.muted),
              ),
              Text(
                name ?? 'Sama Code',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 22,
                  fontWeight: FontWeight.w800,
                  color: Ink2.strong,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 12),
        Semantics(
          button: true,
          label: s.t('notifications'),
          excludeSemantics: true,
          child: Material(
            color: Colors.white,
            shape: const CircleBorder(),
            child: InkWell(
              customBorder: const CircleBorder(),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => const NotificationsView()),
              ),
              child: const SizedBox(
                width: 52,
                height: 52,
                child: Icon(
                  Icons.notifications_none_rounded,
                  size: 28,
                  color: Ink2.strong,
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}

/// Bandeau d'accueil vert : ce que propose l'application.
class _HeroBanner extends StatelessWidget {
  const _HeroBanner({required this.strings});

  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    return Container(
      padding: const EdgeInsets.fromLTRB(22, 22, 12, 22),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(24),
        gradient: const LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: [AppColors.green, AppColors.greenDark],
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  s.t('hero_title'),
                  style: const TextStyle(
                    fontSize: 22,
                    fontWeight: FontWeight.w800,
                    color: Colors.white,
                    height: 1.2,
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  s.t('hero_subtitle'),
                  style: const TextStyle(fontSize: 15, color: Colors.white70),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Icon(
            Icons.traffic_rounded,
            size: 72,
            color: Colors.white.withValues(alpha: 0.9),
          ),
        ],
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  const _SectionTitle(this.title, {this.subtitle});

  final String title;
  final String? subtitle;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(left: 4, bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: const TextStyle(
              fontSize: 21,
              fontWeight: FontWeight.w800,
              color: Ink2.strong,
            ),
          ),
          if (subtitle != null)
            Text(
              subtitle!,
              style: const TextStyle(fontSize: 16, color: Ink2.muted),
            ),
        ],
      ),
    );
  }
}

class _FeatureTile extends StatelessWidget {
  const _FeatureTile({
    required this.icon,
    required this.label,
    required this.badge,
    required this.color,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final String badge;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      button: true,
      label: '$label. $badge',
      excludeSemantics: true,
      child: SoftCard(
        onTap: onTap,
        padding: const EdgeInsets.all(18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 56,
              height: 56,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(16),
              ),
              child: Icon(icon, size: 32, color: color),
            ),
            const SizedBox(height: 14),
            Text(
              label,
              style: const TextStyle(
                fontSize: 19,
                fontWeight: FontWeight.w700,
                color: Ink2.strong,
              ),
            ),
            const SizedBox(height: 2),
            Text(
              badge,
              style: const TextStyle(fontSize: 14, color: Ink2.muted),
            ),
          ],
        ),
      ),
    );
  }
}

/// Carte d'un permis : icône, nom et flèche.
class _LicenseCard extends StatelessWidget {
  const _LicenseCard({
    required this.category,
    required this.strings,
    required this.onTap,
  });

  final LicenseCategory category;
  final AppStrings strings;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final isTruck = category.code == 'C';
    final color = isTruck ? AppColors.orange : AppColors.green;
    // Sous-titre par code de permis (à déplacer en base, traduit, en phase 3).
    final key = 'license_${category.code}_subtitle';
    final subtitle = strings.t(key) == key ? null : strings.t(key);

    return Semantics(
      button: true,
      label: subtitle == null ? category.name : '${category.name}. $subtitle',
      excludeSemantics: true,
      child: SoftCard(
        onTap: onTap,
        padding: const EdgeInsets.all(18),
        child: Row(
          children: [
            Container(
              width: 60,
              height: 60,
              decoration: BoxDecoration(
                color: color,
                borderRadius: BorderRadius.circular(18),
              ),
              child: Icon(
                isTruck
                    ? Icons.local_shipping_rounded
                    : Icons.directions_car_rounded,
                size: 34,
                color: Colors.white,
              ),
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    category.name,
                    style: const TextStyle(
                      fontSize: 22,
                      fontWeight: FontWeight.w800,
                      color: Ink2.strong,
                    ),
                  ),
                  if (subtitle != null)
                    Text(
                      subtitle,
                      style: const TextStyle(fontSize: 16, color: Ink2.muted),
                    ),
                ],
              ),
            ),
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: color.withValues(alpha: 0.12),
                shape: BoxShape.circle,
              ),
              child: Icon(Icons.arrow_forward_rounded, color: color, size: 26),
            ),
          ],
        ),
      ),
    );
  }
}

/// Bandeau « Examen en cours » : reprendre un examen interrompu (chronomètre actif).
class _ResumeBanner extends StatelessWidget {
  const _ResumeBanner({
    required this.exam,
    required this.strings,
    required this.onTap,
  });

  final ExamSession exam;
  final AppStrings strings;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final s = strings;
    return Semantics(
      button: true,
      label: '${s.t('resume_banner')}. ${s.t('resume_banner_desc')}',
      excludeSemantics: true,
      child: Material(
        color: AppColors.orange,
        borderRadius: BorderRadius.circular(24),
        child: InkWell(
          borderRadius: BorderRadius.circular(24),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Row(
              children: [
                const Icon(Icons.timer_outlined, color: Colors.white, size: 44),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        s.t('resume_banner'),
                        style: const TextStyle(
                          fontSize: 21,
                          fontWeight: FontWeight.w800,
                          color: Colors.white,
                        ),
                      ),
                      Text(
                        'Permis ${exam.licenseCode} · ${s.t('series_n')} ${exam.seriesNumber} · ${s.t('resume_banner_desc')}',
                        style: const TextStyle(
                          fontSize: 15,
                          color: Colors.white,
                        ),
                      ),
                    ],
                  ),
                ),
                const Icon(
                  Icons.play_circle_fill_rounded,
                  color: Colors.white,
                  size: 40,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

---
slug: tenant-configuration
title: "Tenant Configuration"
type: process-description
status: draft
created: 2026-06-09
related_requirements: [FR-BB13, FR-BB112, FR-BB62]
---

## Business Goal

Each organisation deploying BilimBaga needs the platform to reflect its own identity — company name, logo, and brand colours — and to support the languages spoken by its employees. This process covers initial platform setup after deployment and ongoing branding maintenance. Success means every employee sees the organisation's branding immediately on the login page (before authentication), and the platform operates in the languages configured by the admin.

## Actors

| Actor | Role |
|-------|------|
| Super Admin | The only role that can change tenant configuration |
| Employee | Experiences branding and language settings; cannot change them |

## Process Steps

### Step 1 — Initial Brand Setup (Super Admin)

1. Super Admin logs in for the first time after deployment.
2. Super Admin navigates to Settings → Branding.
3. Super Admin sets:
   - **Application name**: displayed in the browser title and on the login page header.
   - **Logo**: uploaded as an image file (drag-and-drop or file picker). The system stores the image and serves it to all users.
   - **Primary colour**: the main brand colour used for buttons, active states, and headers.
   - **Accent colour**: a secondary colour used for highlights and links.
4. A live preview panel shows how the logo and colours look on a sample header before saving.
5. The WCAG AA contrast checker warns if the chosen colours will fail accessibility contrast requirements.
6. Super Admin clicks "Save".
7. **Expected outcome**: All users (including unauthenticated visitors on the login page) immediately see the new branding without a page reload being required on subsequent visits.

### Step 2 — Configure Languages (Super Admin)

1. Super Admin navigates to Settings → Branding (or Settings → Localisation).
2. Super Admin selects the available locales for the platform from the configured list (e.g. English, Russian, Kazakh).
3. Super Admin designates one locale as the default locale.
4. Super Admin saves.
5. **Expected outcome**: The language selector on the login page and in user settings shows only the enabled locales. All new questions created must have a translation in the default locale.

### Step 3 — Update Branding (Super Admin)

1. Super Admin navigates to Settings → Branding at any point.
2. Super Admin changes any combination of: application name, logo, primary colour, accent colour.
3. Super Admin saves.
4. The updated branding is reflected immediately for all active users on their next page navigation.
5. Certificates generated before the change retain their original branding (the template snapshot is frozen at generation time).
6. **Expected outcome**: Branding changes are live immediately; historical certificates are not retroactively altered.

## Business Rules

- Only Super Admin can modify tenant configuration; no other role has access to the Settings → Branding page.
- Branding is served to unauthenticated users (login page loads branding before login to ensure correct colours and logo are visible).
- Tenant config is cached in the Go process memory; a save operation invalidates the cache immediately.
- The logo is stored as a binary blob; re-uploading replaces the previous logo; there is no logo version history.
- If the primary or accent colour fails WCAG AA contrast against white (#FFFFFF), a warning is shown but saving is not blocked.
- The default locale cannot be removed from the available locales list.
- Removing a locale from the available list does not delete existing translations in that locale; it only hides the locale from the language selector.

## Acceptance Criteria (business language)

1. After saving a new logo, the login page displays the new logo for a user who loads it fresh (without cache).
2. After saving new primary and accent colours, the platform UI reflects the new colours for all users on their next navigation.
3. The WCAG contrast warning is visible when a colour combination fails AA requirements; the save button remains enabled (warning only, not a block).
4. After changing the default locale, newly created questions must have a translation in the new default locale before being submitted for review.
5. A certificate downloaded before a branding change retains the old logo and colours; a certificate downloaded after the change uses the new branding.
6. A Department Admin does not see the Branding settings page; only Super Admin can access it.

## Out of Scope

- Multi-tenant support (one BilimBaga instance = one organisation; per-tenant schema separation is planned but not in the current scope).
- Custom CSS injection or theme editor beyond logo/colour settings.
- Favicon configuration.
- Email signature or email template branding (handled in the Email Notifications process).

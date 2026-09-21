import React from 'react';
import LegalLayout from './LegalLayout';

const PrivacyPolicy = () => (
  <LegalLayout title="Privacy Policy" updatedAt="2026-09-21">
    <section>
      <h2>1. Data we collect</h2>
      <ul>
        <li><strong>Account data:</strong> your login, display name and profile photo from your identity provider (e.g. 42).</li>
        <li><strong>Session data:</strong> session identifiers, IP address, user agent and last-seen time, used to keep you signed in and to secure your account.</li>
        <li>
          <strong>Module activity data:</strong> for every module you use (via direct access or iframe), we record your
          user ID, the module, and timestamps of usage. See "Module usage activity tracking" in the Terms of Use for how
          this is grouped into activities.
        </li>
      </ul>
    </section>

    <section>
      <h2>2. Why we collect it</h2>
      <ul>
        <li>To authenticate you and keep your session secure.</li>
        <li>To operate module access (routing, permissions).</li>
        <li>To measure module usage: how many people use a module, when usage peaks, and how long modules are used, so administrators can plan capacity and prioritize maintenance.</li>
        <li>To let administrators investigate a specific user's usage of a specific module when needed (e.g. support, abuse investigation, license/seat audits).</li>
      </ul>
    </section>

    <section>
      <h2>3. Who can access this data</h2>
      <p>
        Module activity statistics (including per-user activity) are visible to Platform administrators only, from the
        Stats page in the admin panel. This data is not shared with module owners or third parties beyond what is
        necessary to operate the Platform.
      </p>
    </section>

    <section>
      <h2>4. Retention</h2>
      <p>
        Account and session data is kept for as long as your account exists. Module activity data is kept to preserve
        historical usage trends; administrators may configure a shorter retention period.
      </p>
    </section>

    <section>
      <h2>5. Your rights</h2>
      <p>
        You can delete your account at any time from the Settings page, which removes your account and associated
        session data. Contact a Platform administrator to request access to, or deletion of, your module activity data.
      </p>
    </section>
  </LegalLayout>
);

export default PrivacyPolicy;

import React from 'react';
import LegalLayout from './LegalLayout';

const TermsOfUse = () => (
  <LegalLayout title="Terms of Use" updatedAt="2026-09-21">
    <section>
      <h2>1. Scope</h2>
      <p>
        These Terms of Use govern access to Pan Bagnat ("the Platform"), an internal tool that lets authenticated users
        access modules (internal web applications) either directly or embedded in an iframe from the Platform's dashboard.
      </p>
    </section>

    <section>
      <h2>2. Account &amp; authentication</h2>
      <p>
        Access to the Platform requires signing in with your 42 account (or another identity provider configured by your
        organization). By signing in you agree to these Terms and to the Privacy Policy.
      </p>
    </section>

    <section>
      <h2>3. Module usage activity tracking</h2>
      <p>
        To operate and improve the Platform, Pan Bagnat records <strong>module usage activity</strong>: which module you
        use, when, and for how long, whether you access the module directly or through an iframe embedded in the
        Platform. This tracking is <strong>not anonymous</strong> — activity is attributed to your account and can be
        reviewed by Platform administrators.
      </p>
      <p>Concretely:</p>
      <ul>
        <li>Every request you make to a module while authenticated is logged with your user ID, the module, and a timestamp.</li>
        <li>
          Continuous usage is grouped into "activities": usage is considered part of the same activity as long as there is
          no gap of 60 minutes or more without any request to that module; a gap of 60 minutes or more starts a new
          activity.
        </li>
        <li>Administrators can see aggregate statistics (usage over time, peak usage hours) as well as per-user activity for each module.</li>
      </ul>
      <p>
        This data is used to understand module adoption, detect usage peaks, and help administrators size and maintain
        modules. See the Privacy Policy for retention and access details.
      </p>
    </section>

    <section>
      <h2>4. Acceptable use</h2>
      <p>
        You agree to use the Platform and its modules only for their intended purpose, to comply with your organization's
        internal policies, and not to attempt to bypass access controls or disrupt the Platform or its modules.
      </p>
    </section>

    <section>
      <h2>5. Availability</h2>
      <p>
        Modules are provided "as is" by their respective owners. The Platform administrators do not guarantee
        uninterrupted availability of any module.
      </p>
    </section>

    <section>
      <h2>6. Changes</h2>
      <p>These Terms may be updated from time to time. Continued use of the Platform after a change constitutes acceptance of the updated Terms.</p>
    </section>
  </LegalLayout>
);

export default TermsOfUse;

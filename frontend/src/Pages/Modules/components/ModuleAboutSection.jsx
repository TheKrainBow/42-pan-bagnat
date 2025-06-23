import Link from 'Global/Link';
import './ModuleAboutSection.css';

const ModuleAboutSection = ({ module }) => {
  const isCloned = new Date(module.last_update).getFullYear() > 2000;

  return (
    <div className="module-version-section">
      <div>
        <div>
          <strong>📦 Version:</strong>{' '}
          {isCloned ? module.version : <em>Waiting for clone</em>}
        </div>
        <div>
          <strong>🔄 Latest:</strong>{' '}
          {isCloned ? module.latest_version : <em>Waiting for clone</em>}
        </div>
        <div>
          <strong>🧱 Late Commits:</strong>{' '}
          {isCloned ? module.late_commits : <em>Waiting for clone</em>}
        </div>
        <div>
          <strong>🕒 Last Update:</strong>{' '}
          {isCloned
            ? new Date(module.last_update).toLocaleString()
            : <em>Waiting for clone</em>}
        </div>
        <div>
          <strong>🔗 Repo:</strong> <Link url={module.git_url} />
        </div>
        <div>
          <strong>🔑 SSH Key:</strong>{' '}
          <Link url={module.ssh_public_key} shorten={42} />
        </div>
      </div>
    </div>
  );
};

export default ModuleAboutSection;

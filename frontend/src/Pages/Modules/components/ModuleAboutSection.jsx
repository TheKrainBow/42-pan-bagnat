import { useState } from "react";
import Link from 'Global/Link';
import './ModuleAboutSection.css';

const ModuleAboutSection = ({ module }) => {
const [copied, setCopied] = useState(false);

const handleCopy = () => {
	navigator.clipboard.writeText(module.ssh_public_key);
	setCopied(true);
	setTimeout(() => setCopied(false), 1500);
};

return (
	<div className="module-version-section">
		<div className="version-info">
		<div><strong>📦 Version:</strong> {module.version}</div>
		<div><strong>🔄 Latest:</strong> {module.latest_version}</div>
		<div><strong>🧱 Late Commits:</strong> {module.late_commits}</div>
		<div><strong>🕒 Last Update:</strong> {new Date(module.last_update).toLocaleString()}</div>
		<div><strong>🔗 Repo:</strong> <Link url={module.url} /></div>
		<div><strong>🔑 SSH Key:</strong> <Link url={module.ssh_public_key}  shorten={42}/></div>
		</div>
	</div>
);
}

export default ModuleAboutSection;

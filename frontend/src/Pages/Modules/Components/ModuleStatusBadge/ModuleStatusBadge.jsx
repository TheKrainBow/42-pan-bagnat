import './ModuleStatusBadge.css';

const formatStatus = (status) => {
  if (!status) return '';
  return status.replace(/_/g, ' ').toUpperCase();
};

const ModuleStatusBadge = ({ status }) => {
  return (
    <span className={`status-badge ${status}`}>
      {formatStatus(status)}
    </span>
  );
};

export default ModuleStatusBadge;

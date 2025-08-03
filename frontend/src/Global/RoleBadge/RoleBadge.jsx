import { getReadableStyles } from 'Global/utils/ColorUtils';
import './RoleBadge.css';

const RoleBadge = ({ hexColor, children, onDelete }) => {
  const styles = getReadableStyles(hexColor);

  return (
    <span className="role-badge" style={styles}>
      {children}
      {onDelete && (
        <button className="role-badge-delete" onClick={onDelete}>
          ✕
        </button>
      )}
    </span>
  );
};

export default RoleBadge;

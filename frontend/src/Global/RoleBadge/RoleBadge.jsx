import { getReadableStyles } from '../../utils/ColorUtils';
import './RoleBadge.css';

const RoleBadge = ({ hexColor, children }) => {
  const styles = getReadableStyles(hexColor);
  return (
    <span className="role-badge" style={styles}>
      {children}
    </span>
  );
}

export default RoleBadge;
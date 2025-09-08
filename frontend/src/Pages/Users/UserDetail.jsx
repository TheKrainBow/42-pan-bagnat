import { useEffect, useState, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { fetchWithAuth } from "Global/utils/Auth";
import RoleBadge from "ui/molecules/Badges/RoleBadge";
import ModuleBadge from "ui/molecules/Badges/ModuleBadge";
import { toast } from 'react-toastify';
import './UserDetail.css';

const ADMIN_ROLE_ID = 'roles_admin';

export default function UserDetail() {
  const { identifier } = useParams();
  const [user, setUser] = useState(null);
  const [modules, setModules] = useState([]);
  const [availableRoles, setAvailableRoles] = useState([]);
  const [email, setEmail] = useState("");
  const [isAdmin, setIsAdmin] = useState(false);
  const [showRoleSearch, setShowRoleSearch] = useState(false);
  const [searchRoleTerm, setSearchRoleTerm] = useState("");
  const dropdownRef = useRef();
  const navigate = useNavigate();


  const reloadModules = async (roles) => {
    const moduleMap = new Map();
    for (const role of roles || []) {
      try {
        const res = await fetchWithAuth(`/api/v1/admin/roles/${role.id}`);
        const roleData = await res.json();
        (roleData.modules || []).forEach(mod => {
          moduleMap.set(mod.id, mod); // dedup
        });
      } catch (err) {
        console.error(`Failed to fetch role ${role.id}`, err);
      }
    }
    setModules(Array.from(moduleMap.values()));
  };

  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setShowRoleSearch(false);
      }
    }

    if (showRoleSearch) {
      document.addEventListener("mousedown", handleClickOutside);
    } else {
      document.removeEventListener("mousedown", handleClickOutside);
    }

    return () => {
      document.removeEventListener("mousedown", handleClickOutside);
    };
  }, [showRoleSearch]);

  const filteredRoles = availableRoles.filter((r) =>
    r.id !== ADMIN_ROLE_ID &&                               // hide admin from list
    r.name.toLowerCase().includes(searchRoleTerm.toLowerCase()) &&
    !user?.roles?.some(existing => existing.id === r.id)
  );

  // Initial fetch
  useEffect(() => {
    fetchWithAuth(`/api/v1/admin/roles`)
      .then(res => res.json())
      .then(data => setAvailableRoles(data.roles || []));

    fetchWithAuth(`/api/v1/admin/users/${identifier}`)
      .then(res => res.json())
      .then(async data => {
        data.roles ??= [];
        setUser(data);
        setIsAdmin(data.roles.some(r => r.id === ADMIN_ROLE_ID));
        setEmail(data.email || "");
        await reloadModules(data.roles);
      });
  }, [identifier]);

  async function handleAdminToggle(e) {
    const newValue = e.target.checked;
    setIsAdmin(newValue); // optimistic
    try {
      const url = `/api/v1/admin/users/${identifier}/roles/${ADMIN_ROLE_ID}`;
      const res = await fetchWithAuth(url, {
        method: newValue ? "POST" : "DELETE",
        headers: { "Content-Type": "application/json" },
      });
      if (!res || !res.ok) throw new Error("Failed to toggle admin role");

      setUser(prev => {
        const hasIt = prev.roles?.some(r => r.id === ADMIN_ROLE_ID);
        let nextRoles = prev.roles || [];
        if (newValue && !hasIt) {
          nextRoles = [...nextRoles, { id: ADMIN_ROLE_ID, name: "Pan Bagnat Admin", color: "#1F2937" }];
        } else if (!newValue && hasIt) {
          nextRoles = nextRoles.filter(r => r.id !== ADMIN_ROLE_ID);
        }
        reloadModules(nextRoles);
        return { ...prev, roles: nextRoles };
      });
    } catch (err) {
      console.error(err);
      // revert optimistic state on failure
      setIsAdmin(prev => !prev);
    }
  }

  async function handleAddRole(role) {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/users/${identifier}/roles/${role.id}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
      });

      if (!res.ok) throw new Error("Failed to add role");

      setUser(prev => {
        const updated = [...prev.roles, role];
        // Recalculate filteredRoles manually
        const remaining = availableRoles.filter((r) =>
          r.name.toLowerCase().includes(searchRoleTerm.toLowerCase()) &&
          !updated.some(existing => existing.id === r.id)
        );

        if (remaining.length === 0) {
          setShowRoleSearch(false);
          setSearchRoleTerm("");
        }
        reloadModules(updated);
        return {
          ...prev,
          roles: updated,
        };
      });
    } catch (err) {
      console.error(err);
    }
  }

  async function handleRoleRemove(role) {
    try {
      const res = await fetchWithAuth(`/api/v1/admin/users/${identifier}/roles/${role.id}`, {
        method: "DELETE",
      });

      if (!res.ok) throw new Error("Failed to remove role");

      setUser((prev) => {
        const updatedRoles = prev.roles.filter((r) => r.id !== role.id);
        reloadModules(updatedRoles);
        return {
          ...prev,
          roles: updatedRoles,
        };
      });
    } catch (err) {
      console.error(err);
    }
  }

  if (!user) return <div>Loading...</div>;

  return (
    <div className="user-detail-wrapper">
      {/* Header / Banner */}
      <div className="user-profile-header">
        <img className="avatar" src={user.ft_photo} alt="avatar" />

        <div className="user-profile-header-info">
          <h1>{user.ft_login}</h1>

          <div className="user-meta-grid">
            <div className="label">ID:</div>
            <div className="value">{user.ft_id}</div>

            <div className="label">Is Staff:</div>
            <div className="value">
              <input
                type="checkbox"
                checked={isAdmin}
                onChange={handleAdminToggle}
              />
            </div>
  
            <div className="label">Last Seen:</div>
            <div className="value">{new Date(user.last_seen).toLocaleString()}</div>
          </div>
        </div>
      </div>

      {/* Two Column Body */}
      <div className="user-profile-body">
        <div className="user-modules-panel">
          <div className="role-panel-header">
            <label>Roles:</label>
            <button className="add-role-btn" onClick={() => setShowRoleSearch(!showRoleSearch)}>
              + Add Role
            </button>
          </div>

          <div className="user-role-tags">
            {(user.roles.filter(r => r.id !== ADMIN_ROLE_ID)).length === 0 ? (
              <i>No roles assigned</i>
            ) : (
              user.roles
                .filter(r => r.id !== ADMIN_ROLE_ID) // hide admin badge
                .map((role) => (
                  <RoleBadge
                    key={role.id}
                    role={role}
                    onClick={() => navigate(`/admin/roles/${role.id}`)}
                    onDelete={() => handleRoleRemove(role)}
                  >
                    {role.name}
                  </RoleBadge>
                ))
            )}
          </div>

          {showRoleSearch && (
            <div className="role-search-dropdown" ref ={dropdownRef}>
              <input
                type="text"
                placeholder="Search roles..."
                value={searchRoleTerm}
                onChange={(e) => setSearchRoleTerm(e.target.value)}
              />
              <ul className="role-search-list">
                {filteredRoles.length === 0 ? (
                  <li className="role-line no-result">No more roles available</li>
                ) : (
                  filteredRoles.map((role) => (
                    <li key={role.id}>
                      <div
                        className="role-line role-line-clickable"
                        onClick={() => handleAddRole(role)}
                      >
                        <RoleBadge role={role}>{role.name}</RoleBadge>
                      </div>
                    </li>
                  ))
                )}
              </ul>
            </div>
          )}
        </div>
      </div>


      <div className="user-profile-body">
        <div className="user-modules-panel">
          <label>Accessible Modules:</label>
          <div className="modules-grid">
            {modules.length === 0 ? (
              <i>No modules available</i>
            ) : (
              modules.map(mod => (
                <ModuleBadge key={mod.id} mod={mod} />
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

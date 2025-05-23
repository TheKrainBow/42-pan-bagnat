import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';

const ModuleDetails = () => {
  const { moduleId } = useParams();
  const [module, setModule] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchModule = async () => {
      try {
        const res = await fetch(`http://localhost:8080/api/v1/modules/${moduleId}`);
        const data = await res.json();
        setModule(data);
      } catch (err) {
        console.error(err);
        setModule(null);
      } finally {
        setLoading(false);
      }
    };

    fetchModule();
  }, [moduleId]);

  if (loading) return <div className="loading">Loading...</div>;
  if (!module) return <div className="error">Module not found.</div>;

  return (
    <div className="module-details">
      <h2>{module.name}</h2>
      <pre>{JSON.stringify(module, null, 2)}</pre>
    </div>
  );
};

export default ModuleDetails;

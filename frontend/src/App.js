import React, { useEffect, useState } from 'react';
import './App.css';

function App() {
  const [version, setVersion] = useState(null);

  useEffect(() => {
    // Make the API call to get the version
    fetch('http://localhost:8080/api/version')
      .then((response) => response.json())
      .then((data) => {
        setVersion(data.version); // Store the version in state
      })
      .catch((error) => {
        console.error('Error fetching version:', error);
      });
  }, []); // Empty dependency array ensures this runs only once when the component mounts

  return (
    <div className="App">
      <header className="App-header">
        {version && (
          <div>
            <h1>Backend Version: {version}</h1>
          </div>
        )}
        <p>
          Edit <code>src/App.js</code> and save to reload.
        </p>
        <a
          className="App-link"
          href="https://reactjs.org"
          target="_blank"
          rel="noopener noreferrer"
        >
          Learn React
        </a>
      </header>
    </div>
  );
}

export default App;

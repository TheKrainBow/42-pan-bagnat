import { useState, useEffect } from 'react';

export function AppIcone({ app, fallback }) {
    const [status, setStatus] = useState('loading'); // 'loading', 'loaded', 'error'
    const [src, setSrc] = useState(fallback);
  
    useEffect(() => {
      const image = new Image();
  
      image.onload = () => {
        setSrc(app.icone_url);
        setStatus('loaded');
      };
  
      image.onerror = () => {
        setSrc(fallback);
        setStatus('error');
      };
  
      if (app.icone_url && app.icone_url.trim() !== '') {
        image.src = app.icone_url;
      } else {
        setStatus('error');
      }
    }, [app.icone_url, fallback]);
  
    return (
        <div
          style={{
            width: 48,
            height: 48,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          {status === 'loading' ? (
            <div className="loader" style={{ width: 24, height: 24 }} />
          ) : (
            <img
              src={src}
              alt={app.name}
              title={app.name}
              style={{
                height: 48,
                width: 48,
                maxWidth: 48,
                marginRight: 4,
              }}
            />
          )}
        </div>
    );
  }
// src/Components/LoginCard.jsx
import React, { useState, useRef } from "react";
import "./LoginCard.css";
import Button from "Global/Button/Button";
import Field from "Global/Field/Field";
import { toast } from 'react-toastify';

export default function LoginCard({ onLogin, onMagicLink }) {
  const [downMode, setDownMode] = useState(false);
  const [login, setEmail] = useState("");
  const loginFieldRef = useRef(null);

  const toggleDownMode = () => setDownMode(prev => !prev);

  const handleMagicLink = () => {
	const isValid = loginFieldRef.current.isValid(true);
    if (!isValid) {
		loginFieldRef.current.triggerShake();
		return
	}
	toast.success("Nothing happened but let's pretend it's working")
  };
  return (
    <div className="login-card">
      <div className="card-header">
        <img src="/icons/panbagnat.png" alt="Pan Bagnat Logo" className="card-logo" />
        <h1>Pan Bagnat</h1>
      </div>

      {!downMode ? (
        <>
		<div className="login-42-btn">
			<Button label="Sign in with 42 Intranet" onClick={onLogin} icon="/icons/42.svg"/>
		</div>
          <button className="down-link" onClick={toggleDownMode}>
            42 Intranet is down?
          </button>
        </>
      ) : (
        <>
          <div className="magic-form">
            <Field
              ref={loginFieldRef}
              label="Login"
              value={login}
              onChange={e => setEmail(e.target.value)}
              placeholder="maagosti"
              required={true}
            />
			<Button
				label="Send Magic Link"
				onClick={handleMagicLink}
				disabled={true}
				disabledMessage={"Not implemented (whoops)"}
			/>
          </div>
          <button className="down-link" onClick={toggleDownMode}>
            Nevermind!
          </button>
        </>
      )}
    </div>
  );
}
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import styles from './Login.module.css';

type Panel = 'login' | 'register';

interface LoginForm {
  email: string;
  password: string;
}

interface RegisterForm {
  username: string;
  email: string;
  password: string;
  confirmPassword: string;
}

export default function Login() {
  const navigate = useNavigate();
  const [panel, setPanel] = useState<Panel>('login');
  const [error, setError] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);

  const getCookies = (name:string) => {
      console.log(document.cookie)
      if (document.cookie == ""){
        return false
      } else {
        const value = document.cookie.split("=")[1]
        return value
      }
  }
  const [loginForm, setLoginForm] = useState<LoginForm>({
    email: '',
    password: '',
  });

  const [registerForm, setRegisterForm] = useState<RegisterForm>({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  });

  // ── Login ────────────────────────────────────────────────────────────────
  async function handleLogin(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const res = await fetch('http://localhost:4000/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify(loginForm),
      });
      if (!res.ok) {
        const msg = await res.text();
        setError(msg || 'Invalid username or password');
        return;
      }
      navigate('/accountHome');
    } catch {
      setError('Could not reach the server. Try again later.');
    } finally {
      setLoading(false);
    }
  }

  // ── Register ─────────────────────────────────────────────────────────────
  async function handleRegister(e: React.FormEvent) {
    e.preventDefault();
    setError('');

    if (registerForm.password !== registerForm.confirmPassword) {
      setError('Passwords do not match');
      return;
    }

    setLoading(true);
    try {
      const res = await fetch('http://localhost:4000/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          username: registerForm.username,
          email: registerForm.email,
          password: registerForm.password,
        }),
      });
      if (!res.ok) {
        const msg = await res.text();
        setError(msg || 'Registration failed');
        return;
      }
      navigate('/accountHome');
    } catch {
      setError('Could not reach the server. Try again later.');
    } finally {
      setLoading(false);
    }
  }

  // ── Guest ────────────────────────────────────────────────────────────────
  async function handleGuest() {
    setError('');
    setLoading(true);
    try {
      const res = await fetch('http://localhost:4000/auth/guest', {
        method: 'POST',
        credentials: 'include',
      });
      if (!res.ok) {
        setError('Could not start guest session');
        return;
      }
      navigate('/demo');
    } catch {
      setError('Could not reach the server. Try again later.');
    } finally {
      setLoading(false);
    }
  }
  // useEffect(() => {
  //   const new_account = getCookies("new_account")
  //   if (new_account){
  //     setPanel('register')
  //   } else {
  //     setPanel('login')
  //   }
  // },[])
  return (
    <div className={styles.page}>
      <div className={styles.viewport}>
        <div className={`${styles.slider} ${panel === 'register' ? styles.sliderShifted : ''}`}>

          {/* ── Login Panel ─────────────────────────────────────────────── */}
          <div className={styles.panel}>
            <div className={styles.card}>
              <div className={styles.logo}>CS2 Stratbook</div>
              <h1 className={styles.title}>Welcome!</h1>
              <p className={styles.sub}>Sign in to your account</p>

              {error && panel === 'login' && (
                <div className={styles.error}>{error}</div>
              )}

              {/* Steam */}
              <a href="http://localhost:4000/auth/steam" className={styles.steamBtn}>
                <svg className={styles.steamIcon} viewBox="0 0 496 512" fill="currentColor">
                  <path d="M496 256c0 137-111.2 248-248.4 248-113.8 0-209.7-76.3-239.1-180.4l95.2 39.3c6.4 32.1 35.2 56.4 70.3 56.4 39.2 0 71.1-32.1 71.1-71.6 0-2.9-.2-5.8-.6-8.6l83.5-66.8c22.8 1.4 45.2-5.8 62.1-21.8 29.7-28.2 30.7-74.8 2.5-104.4-28.2-29.7-74.8-30.7-104.4-2.5-17 16-25.5 38-24.8 60l-65.7 93.8c-17.4-1.7-34.8 3.8-47.9 16.3L19.7 282.3C34.2 385.5 123.7 464 231.6 464 348.2 464 443 369.2 443 252.7c0-116.5-94.8-211.3-211.4-211.3-37.8 0-73.2 9.9-103.9 27.4l-40.8-16.8c38.4-29.1 86-46.3 137.7-46.3C384.8 5.7 496 116.9 496 256z"/>
                </svg>
                Sign in through Steam
              </a>

              <div className={styles.divider}><span>or</span></div>

              {/* Username / Password */}
              <form onSubmit={handleLogin} className={styles.form}>
                <input
                  className={styles.input}
                  type="text"
                  placeholder="Username"
                  value={loginForm.email}
                  onChange={e => setLoginForm({ ...loginForm, email: e.target.value })}
                  required
                />
                <input
                  className={styles.input}
                  type="password"
                  placeholder="Password"
                  value={loginForm.password}
                  onChange={e => setLoginForm({ ...loginForm, password: e.target.value })}
                  required
                />
                <div className={styles.btnRow}>
                    <button className={styles.primaryBtn} type="submit" disabled={loading}>
                        {loading ? 'Signing in...' : 'Sign in'}
                    </button>
                    <button className={styles.guestBtn} onClick={handleGuest} disabled={loading}>
                        Continue as Guest
                    </button>
                    </div>
                
              </form>

              {/* Guest */}
              

              <p className={styles.switchText}>
                Don't have an account?{' '}
                <button className={styles.switchLink} onClick={() => { setError(''); setPanel('register'); }}>
                  Create one
                </button>
              </p>
            </div>
          </div>

          {/* ── Register Panel ───────────────────────────────────────────── */}
          <div className={styles.panel}>
            <div className={styles.card}>
              <div className={styles.logo}>GoCS</div>
              <h1 className={styles.title}>Create account</h1>
              <p className={styles.sub}>Get started with GoCS</p>

              {error && panel === 'register' && (
                <div className={styles.error}>{error}</div>
              )}

              <form onSubmit={handleRegister} className={styles.form}>
                <input
                  className={styles.input}
                  type="text"
                  placeholder="Username"
                  value={registerForm.username}
                  onChange={e => setRegisterForm({ ...registerForm, username: e.target.value })}
                  required
                />
                <input
                  className={styles.input}
                  type="email"
                  placeholder="Email"
                  value={registerForm.email}
                  onChange={e => setRegisterForm({ ...registerForm, email: e.target.value })}
                  required
                />
                <input
                  className={styles.input}
                  type="password"
                  placeholder="Password"
                  value={registerForm.password}
                  onChange={e => setRegisterForm({ ...registerForm, password: e.target.value })}
                  required
                />
                <input
                  className={styles.input}
                  type="password"
                  placeholder="Confirm password"
                  value={registerForm.confirmPassword}
                  onChange={e => setRegisterForm({ ...registerForm, confirmPassword: e.target.value })}
                  required
                />
                <button className={styles.primaryBtn} type="submit" disabled={loading}>
                  {loading ? 'Creating account...' : 'Create account'}
                </button>
              </form>

              <p className={styles.switchText}>
                Already have an account?{' '}
                <button className={styles.switchLink} onClick={() => { setError(''); setPanel('login'); }}>
                  Sign in
                </button>
              </p>
            </div>
          </div>

        </div>
      </div>
    </div>
  );
}
import { Link, useNavigate } from 'react-router-dom';
import styles from '../css/Navbar.module.css';
import { useAuth } from '../context/AuthContext';

export default function Navbar() {
  const { user, loading } = useAuth();
  return (
    <nav className={styles.navbar}>
      <div className={styles.navLinks}>
        <Link to="/accountHome">Home</Link>
        <Link to="/team">Team</Link>
        <Link to="/demoList">Pro Matches</Link>
        <Link to="/whiteboard">Whiteboard</Link>
      </div>
      <div className={styles.navRight}>
          {loading ? (
            <div className={styles.avatarSkeleton} />
          ) : user ? (
            <Link to={`/player/me`} className={styles.avatarLink}>
              <img src={user.profilePic} alt={user.playerName} className={styles.avatar} />
            </Link>
          ) : (
            <a href="http://localhost:4000/auth/steam" className={styles.navSteamBtn}>
                <svg className={styles.navSteamIcon} viewBox="0 0 496 512" fill="currentColor">
                  <path d="M496 256c0 137-111.2 248-248.4 248-113.8 0-209.7-76.3-239.1-180.4l95.2 39.3c6.4 32.1 35.2 56.4 70.3 56.4 39.2 0 71.1-32.1 71.1-71.6 0-2.9-.2-5.8-.6-8.6l83.5-66.8c22.8 1.4 45.2-5.8 62.1-21.8 29.7-28.2 30.7-74.8 2.5-104.4-28.2-29.7-74.8-30.7-104.4-2.5-17 16-25.5 38-24.8 60l-65.7 93.8c-17.4-1.7-34.8 3.8-47.9 16.3L19.7 282.3C34.2 385.5 123.7 464 231.6 464 348.2 464 443 369.2 443 252.7c0-116.5-94.8-211.3-211.4-211.3-37.8 0-73.2 9.9-103.9 27.4l-40.8-16.8c38.4-29.1 86-46.3 137.7-46.3C384.8 5.7 496 116.9 496 256z"/>
                </svg>
                Sign in through Steam
              </a>
          )}
      </div>
    </nav>
  );
}
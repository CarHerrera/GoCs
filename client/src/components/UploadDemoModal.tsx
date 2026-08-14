import React, { useRef, useState, useEffect, type ReactElement } from 'react';
import '../css/SettingsModal.css';

interface Props {
  onClose: () => void;
  onUploaded?: () => void;
}

interface Season {
  id: number;
  name: string;
}

const UploadDemoModal: React.FC<Props> = ({ onClose, onUploaded }): ReactElement => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [matchLabel, setMatchLabel] = useState('');
  const [seasons, setSeasons] = useState<Season[]>([]);
  const [seasonId, setSeasonId] = useState('');
  const [status, setStatus] = useState<{ type: 'success' | 'error'; msg: string } | null>(null);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    fetch('http://localhost:4000/api/team/seasons', { credentials: 'include' })
      .then(r => r.json())
      .then((data: Season[]) => setSeasons(data ?? []))
      .catch(() => {});
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null;
    setFile(f);
    setStatus(null);
  };

  const handleSubmit = async () => {
    if (!file || !seasonId) return;
    setUploading(true);
    setStatus(null);

    const fd = new FormData();
    fd.append('demo', file);
    fd.append('match_label', matchLabel.trim());
    fd.append('season_id', seasonId);

    try {
      const res = await fetch('http://localhost:4000/api/team/upload-demo', {
        method: 'POST',
        credentials: 'include',
        body: fd,
      });
      if (res.ok) {
        setStatus({ type: 'success', msg: 'Demo uploaded successfully.' });
        onUploaded?.();
      } else {
        const text = await res.text();
        setStatus({ type: 'error', msg: text || 'Upload failed.' });
      }
    } catch {
      setStatus({ type: 'error', msg: 'Network error — could not reach server.' });
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="upload-modal-overlay" onClick={onClose}>
      <div className="upload-modal" onClick={e => e.stopPropagation()}>
        <div className="upload-modal-header">
          <h3>Upload Demo</h3>
          <button className="settings-close" onClick={onClose}>✕</button>
        </div>

        <div className="upload-field">
          <label>Demo File (.dem)</label>
          <div className="upload-file-row">
            <span className={`upload-file-name ${file ? 'selected' : ''}`}>
              {file ? file.name : 'No file selected'}
            </span>
            <button
              className="upload-browse-btn"
              onClick={() => fileInputRef.current?.click()}
            >
              Browse
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".dem"
              style={{ display: 'none' }}
              onChange={handleFileChange}
            />
          </div>
        </div>

        <div className="upload-field">
          <label>Match Played</label>
          <input
            className="upload-input"
            placeholder="e.g. vs Team Liquid — Finals"
            value={matchLabel}
            onChange={e => setMatchLabel(e.target.value)}
          />
        </div>

        <div className="upload-field">
          <label>Season</label>
          {seasons.length === 0 ? (
            <p className="upload-status error">
              No seasons yet — create one in Settings before uploading.
            </p>
          ) : (
            <select
              className="upload-input"
              value={seasonId}
              onChange={e => setSeasonId(e.target.value)}
            >
              <option value="">Select a season…</option>
              {seasons.map(s => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
          )}
        </div>

        {status && (
          <p className={`upload-status ${status.type}`}>{status.msg}</p>
        )}

        <div className="upload-actions">
          <button className="upload-cancel-btn" onClick={onClose}>Cancel</button>
          <button
            className="upload-submit-btn"
            onClick={handleSubmit}
            disabled={!file || !seasonId || uploading}
          >
            {uploading ? 'Uploading…' : 'Upload'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default UploadDemoModal;
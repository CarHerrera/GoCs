import React, { useRef, useState, type ReactElement } from 'react';
import '../css/SettingsModal.css';

interface Props {
  onClose: () => void;
  onUploaded?: () => void;
}

const UploadDemoModal: React.FC<Props> = ({ onClose, onUploaded }): ReactElement => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File | null>(null);
  const [matchLabel, setMatchLabel] = useState('');
  const [season, setSeason] = useState('');
  const [status, setStatus] = useState<{ type: 'success' | 'error'; msg: string } | null>(null);
  const [uploading, setUploading] = useState(false);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] ?? null;
    setFile(f);
    setStatus(null);
  };

  const handleSubmit = async () => {
    if (!file) return;
    setUploading(true);
    setStatus(null);

    const fd = new FormData();
    fd.append('demo', file);
    fd.append('match_label', matchLabel.trim());
    fd.append('season', season.trim());

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
          <input
            className="upload-input"
            placeholder="e.g. Spring 2025"
            value={season}
            onChange={e => setSeason(e.target.value)}
          />
        </div>

        {status && (
          <p className={`upload-status ${status.type}`}>{status.msg}</p>
        )}

        <div className="upload-actions">
          <button className="upload-cancel-btn" onClick={onClose}>Cancel</button>
          <button
            className="upload-submit-btn"
            onClick={handleSubmit}
            disabled={!file || uploading}
          >
            {uploading ? 'Uploading…' : 'Upload'}
          </button>
        </div>
      </div>
    </div>
  );
};

export default UploadDemoModal;
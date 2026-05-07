const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

function getToken() {
  return localStorage.getItem('token');
}

export async function apiFetch(path, options = {}) {
  const url = `${API_URL}${path}`;
  const opts = {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(getToken() ? { Authorization: `Bearer ${getToken()}` } : {}),
      ...options.headers,
    },
  };
  const res = await fetch(url, opts);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`);
  }
  return data;
}

export function setToken(token) {
  localStorage.setItem('token', token);
}

export function removeToken() {
  localStorage.removeItem('token');
  localStorage.removeItem('user_id');
  localStorage.removeItem('username');
}

export function setUsername(username) {
  localStorage.setItem('username', username);
}

export function getUsername() {
  return localStorage.getItem('username') || '';
}

export function isLoggedIn() {
  return !!getToken();
}

export function setUserId(id) {
  localStorage.setItem('user_id', String(id));
}

export function getUserId() {
  return localStorage.getItem('user_id');
}

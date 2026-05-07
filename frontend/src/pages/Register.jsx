import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiFetch, setToken, setUserId, setUsername } from '../api'

export default function Register({ onLogin }) {
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()

  async function handleSubmit(e) {
    e.preventDefault()
    setError('')
    try {
      await apiFetch('/register', {
        method: 'POST',
        body: JSON.stringify({ username, email, password }),
      })
      const data = await apiFetch('/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
      setToken(data.token)
      setUserId(data.user_id)
      setUsername(data.username)
      onLogin()
      navigate('/')
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="max-w-md mx-auto bg-white rounded-xl shadow-sm border border-gray-200 p-8 mt-10">
      <h1 className="text-2xl font-bold mb-6 text-center">Create your account</h1>
      {error && <div className="mb-4 text-red-600 text-sm bg-red-50 p-3 rounded">{error}</div>}
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
          <input className="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" value={username} onChange={e => setUsername(e.target.value)} required />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
          <input type="email" className="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" value={email} onChange={e => setEmail(e.target.value)} required />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
          <input type="password" className="w-full border border-gray-300 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500" value={password} onChange={e => setPassword(e.target.value)} required />
        </div>
        <button type="submit" className="w-full bg-blue-600 text-white py-2 rounded-full font-medium hover:bg-blue-700 transition">Sign up</button>
      </form>
    </div>
  )
}

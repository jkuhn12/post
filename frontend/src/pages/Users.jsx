import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { apiFetch, getUserId } from '../api'

export default function Users() {
  const [users, setUsers] = useState([])
  const [error, setError] = useState('')
  const [following, setFollowing] = useState(new Set())

  async function loadUsers() {
    try {
      const data = await apiFetch('/users')
      setUsers(data)
    } catch (err) {
      setError(err.message)
    }
  }

  useEffect(() => {
    loadUsers()
  }, [])

  async function handleFollow(id) {
    try {
      // Need current user ID - we don't have it easily without decoding JWT.
      // For simplicity, let's fetch from token or store it. We'll parse it from a simple endpoint or localStorage.
      const userId = getUserId()
      if (!userId) {
        setError('Session error. Please log in again.')
        return
      }
      await apiFetch('/follow', {
        method: 'POST',
        body: JSON.stringify({ follower_id: parseInt(userId), following_id: id }),
      })
      setFollowing(prev => new Set(prev).add(id))
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      <h1 className="text-xl font-bold mb-4">Explore Users</h1>
      {error && <div className="mb-4 text-red-600 text-sm bg-red-50 p-3 rounded">{error}</div>}
      <div className="space-y-3">
        {users.map(user => (
          <div key={user.id} className="bg-white rounded-xl shadow-sm border border-gray-200 p-4 flex items-center justify-between">
            <Link to={`/profile/${user.id}`} className="font-medium text-gray-900 hover:text-blue-600">
              @{user.username}
            </Link>
            <button
              onClick={() => handleFollow(user.id)}
              disabled={following.has(user.id)}
              className={`px-4 py-1.5 rounded-full text-sm font-medium transition ${following.has(user.id) ? 'bg-gray-200 text-gray-600' : 'bg-blue-600 text-white hover:bg-blue-700'}`}
            >
              {following.has(user.id) ? 'Following' : 'Follow'}
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}

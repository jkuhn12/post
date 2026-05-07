import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { apiFetch } from '../api'

export default function Profile() {
  const { id } = useParams()
  const [user, setUser] = useState(null)
  const [posts, setPosts] = useState([])
  const [error, setError] = useState('')

  useEffect(() => {
    async function load() {
      try {
        const u = await apiFetch(`/users/${id}`)
        setUser(u)
        const p = await apiFetch(`/posts?user_id=${id}`)
        setPosts(p)
      } catch (err) {
        setError(err.message)
      }
    }
    load()
  }, [id])

  if (error) return <div className="text-red-600">{error}</div>
  if (!user) return <div className="text-center py-10">Loading...</div>

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6">
        <h1 className="text-2xl font-bold">@{user.user.username}</h1>
        <p className="text-gray-500 text-sm mt-1">{user.user.email}</p>
        <div className="flex gap-6 mt-4 text-sm">
          <div><span className="font-bold">{user.following_count}</span> Following</div>
          <div><span className="font-bold">{user.followers_count}</span> Followers</div>
        </div>
      </div>

      <div>
        <h2 className="text-lg font-bold mb-3">Posts</h2>
        <div className="space-y-4">
          {posts.length === 0 ? (
            <div className="text-gray-500">No posts yet.</div>
          ) : (
            posts.map(post => (
              <div key={post.id} className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
                <div className="flex items-center justify-between mb-2">
                  <span className="font-bold text-gray-900">@{user.user.username}</span>
                  <span className="text-sm text-gray-500">{new Date(post.created_at).toLocaleString()}</span>
                </div>
                <p className="text-gray-800 whitespace-pre-wrap">{post.content}</p>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}

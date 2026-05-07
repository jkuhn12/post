import { useState, useEffect } from 'react'
import { apiFetch, getUsername } from '../api'

export default function Feed() {
  const [posts, setPosts] = useState([])
  const [content, setContent] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  async function loadFeed() {
    setLoading(true)
    try {
      const data = await apiFetch('/feed')
      setPosts(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadFeed()
  }, [])

  async function handlePost(e) {
    e.preventDefault()
    if (!content.trim()) return
    try {
      await apiFetch('/posts', {
        method: 'POST',
        body: JSON.stringify({ content, username: getUsername() }),
      })
      setContent('')
      loadFeed()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div className="space-y-6">
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
        <form onSubmit={handlePost}>
          <textarea
            className="w-full border border-gray-300 rounded-lg px-4 py-3 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
            rows={3}
            placeholder="What's happening?"
            maxLength={320}
            value={content}
            onChange={e => setContent(e.target.value)}
          />
          <div className="flex justify-between items-center mt-2">
            <span className="text-sm text-gray-500">{content.length}/320</span>
            <button type="submit" className="bg-blue-600 text-white px-6 py-2 rounded-full font-medium hover:bg-blue-700 transition">Post</button>
          </div>
        </form>
      </div>

      {error && <div className="text-red-600 text-sm bg-red-50 p-3 rounded">{error}</div>}

      <div className="space-y-4">
        {loading ? (
          <div className="text-center text-gray-500 py-10">Loading...</div>
        ) : posts.length === 0 ? (
          <div className="text-center text-gray-500 py-10">Your feed is empty. Follow some users to see their posts!</div>
        ) : (
          posts.map(post => (
            <div key={post.id} className="bg-white rounded-xl shadow-sm border border-gray-200 p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-gray-900">@{post.username || `User #${post.user_id}`}</span>
                <span className="text-sm text-gray-500">{new Date(post.created_at).toLocaleString()}</span>
              </div>
              <p className="text-gray-800 whitespace-pre-wrap">{post.content}</p>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

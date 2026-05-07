import { Routes, Route, Navigate, Link } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { isLoggedIn, removeToken } from './api'
import Login from './pages/Login'
import Register from './pages/Register'
import Feed from './pages/Feed'
import Users from './pages/Users'
import Profile from './pages/Profile'

function Navbar({ loggedIn, onLogout }) {
  return (
    <nav className="bg-white border-b border-gray-200 sticky top-0 z-10">
      <div className="max-w-2xl mx-auto px-4 py-3 flex items-center justify-between">
        <Link to="/" className="text-xl font-bold text-blue-600">Post</Link>
        <div className="flex gap-4 items-center">
          {loggedIn ? (
            <>
              <Link to="/" className="text-gray-700 hover:text-blue-600 font-medium">Feed</Link>
              <Link to="/users" className="text-gray-700 hover:text-blue-600 font-medium">Users</Link>
              <button onClick={onLogout} className="text-sm text-red-600 hover:underline">Logout</button>
            </>
          ) : (
            <>
              <Link to="/login" className="text-gray-700 hover:text-blue-600 font-medium">Login</Link>
              <Link to="/register" className="bg-blue-600 text-white px-4 py-2 rounded-full text-sm font-medium hover:bg-blue-700">Sign Up</Link>
            </>
          )}
        </div>
      </div>
    </nav>
  )
}

function App() {
  const [loggedIn, setLoggedIn] = useState(isLoggedIn())

  useEffect(() => {
    setLoggedIn(isLoggedIn())
  }, [])

  const handleLogin = () => setLoggedIn(true)
  const handleLogout = () => {
    removeToken()
    setLoggedIn(false)
  }

  return (
    <div className="min-h-screen">
      <Navbar loggedIn={loggedIn} onLogout={handleLogout} />
      <div className="max-w-2xl mx-auto px-4 py-6">
        <Routes>
          <Route path="/login" element={loggedIn ? <Navigate to="/" /> : <Login onLogin={handleLogin} />} />
          <Route path="/register" element={loggedIn ? <Navigate to="/" /> : <Register onLogin={handleLogin} />} />
          <Route path="/" element={loggedIn ? <Feed /> : <Navigate to="/login" />} />
          <Route path="/users" element={loggedIn ? <Users /> : <Navigate to="/login" />} />
          <Route path="/profile/:id" element={loggedIn ? <Profile /> : <Navigate to="/login" />} />
        </Routes>
      </div>
    </div>
  )
}

export default App

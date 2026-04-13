import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
} from 'react-router-dom'
import AssignmentDetailPage from './pages/AssignmentDetailPage'
import AssignmentSubmissionPage from './pages/AssignmentSubmissionPage'
import CourseAnalyticsPage from './pages/CourseAnalyticsPage'
import CourseAssignmentsPage from './pages/CourseAssignmentsPage'
import CourseHomePage from './pages/CourseHomePage'
import CourseMaterialsPage from './pages/CourseMaterialsPage'
import DashboardPage from './pages/DashboardPage'
import LoginPage from './pages/LoginPage'
import SignupPage from './pages/SignupPage'
import { clearAuthSession, isTokenValid } from './utils/auth'

function hasValidSessionToken() {
  const token = localStorage.getItem('token')
  if (!token) {
    return false
  }

  if (!isTokenValid(token)) {
    clearAuthSession()
    return false
  }

  return true
}

function ProtectedRoute({ children }) {
  if (!hasValidSessionToken()) {
    return <Navigate to="/login" replace />
  }
  return children
}

function PublicOnlyRoute({ children }) {
  if (hasValidSessionToken()) {
    return <Navigate to="/dashboard" replace />
  }
  return children
}

function App() {
  const hasToken = hasValidSessionToken()

  return (
    <BrowserRouter>
      <Routes>
        <Route
          path="/"
          element={<Navigate to={hasToken ? '/dashboard' : '/login'} replace />}
        />
        <Route
          path="/login"
          element={
            <PublicOnlyRoute>
              <LoginPage />
            </PublicOnlyRoute>
          }
        />
        <Route
          path="/signup"
          element={
            <PublicOnlyRoute>
              <SignupPage />
            </PublicOnlyRoute>
          }
        />
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <DashboardPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/courses/:courseId"
          element={
            <ProtectedRoute>
              <CourseHomePage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/courses/:courseId/assignments"
          element={
            <ProtectedRoute>
              <CourseAssignmentsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/courses/:courseId/materials"
          element={
            <ProtectedRoute>
              <CourseMaterialsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/courses/:courseId/assignments/:assignmentId"
          element={
            <ProtectedRoute>
              <AssignmentDetailPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/courses/:courseId/assignments/:assignmentId/submission"
          element={
            <ProtectedRoute>
              <AssignmentSubmissionPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/courses/:courseId/analytics"
          element={
            <ProtectedRoute>
              <CourseAnalyticsPage />
            </ProtectedRoute>
          }
        />
      </Routes>
    </BrowserRouter>
  )
}

export default App

import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './auth/AuthContext'
import { Layout } from './components/Layout'
import { Loading } from './components/ui'
import { CourseEditor } from './pages/CourseEditor'
import { Courses } from './pages/Courses'
import { Dashboard } from './pages/Dashboard'
import { LessonEditor } from './pages/LessonEditor'
import { Licenses } from './pages/Licenses'
import { Login } from './pages/Login'
import { QuestionEditor } from './pages/QuestionEditor'
import { Questions } from './pages/Questions'
import { Series } from './pages/Series'
import { SeriesEditor } from './pages/SeriesEditor'
import { Users } from './pages/Users'

export function App() {
  const { me, loading, can } = useAuth()
  if (loading) return <Loading />
  if (!me) return <Login />

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<Dashboard />} />
        <Route path="questions" element={<Questions />} />
        <Route path="questions/new" element={<QuestionEditor />} />
        <Route path="questions/:id" element={<QuestionEditor />} />
        <Route path="series" element={<Series />} />
        <Route path="series/:id" element={<SeriesEditor />} />
        <Route path="courses" element={<Courses />} />
        <Route path="courses/:id" element={<CourseEditor />} />
        <Route path="lessons/:id" element={<LessonEditor />} />
        {can('admin') && <Route path="users" element={<Users />} />}
        {can('admin') && <Route path="licenses" element={<Licenses />} />}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  )
}

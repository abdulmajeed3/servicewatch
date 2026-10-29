import { Link } from 'react-router-dom'
import { Button } from '../components/ui/Button'

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 text-center">
      <h1 className="text-6xl font-bold text-gray-200 dark:text-gray-700">404</h1>
      <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Page not found</h2>
      <p className="text-sm text-gray-500">The page you're looking for doesn't exist.</p>
      <Link to="/dashboard">
        <Button variant="secondary">Go to dashboard</Button>
      </Link>
    </div>
  )
}

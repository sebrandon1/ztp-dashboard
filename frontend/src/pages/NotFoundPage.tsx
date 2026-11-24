import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { Compass } from 'lucide-react';

export default function NotFoundPage() {
  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="card max-w-md w-full p-8 text-center space-y-4"
      >
        <Compass className="w-10 h-10 text-primary-400 mx-auto" />
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Page not found</h1>
          <p className="text-sm text-text-muted mt-2">
            The page you are looking for does not exist or was moved.
          </p>
        </div>
        <Link to="/dashboard" className="btn btn-primary inline-flex">
          Back to Dashboard
        </Link>
      </motion.div>
    </div>
  );
}

import { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, X } from 'lucide-react';

interface ConfirmationModalProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText: string;
  variant: 'danger' | 'warning';
}

export default function ConfirmationModal({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmText,
  variant,
}: ConfirmationModalProps) {
  const [inputValue, setInputValue] = useState('');

  const handleClose = () => {
    setInputValue('');
    onClose();
  };

  const isMatch = inputValue === confirmText;

  const variantStyles = {
    danger: {
      icon: 'text-red-400',
      button: 'bg-red-600 hover:bg-red-700 focus:ring-red-500',
      buttonDisabled: 'bg-red-600/40 cursor-not-allowed',
      border: 'border-red-500/30',
    },
    warning: {
      icon: 'text-amber-400',
      button: 'bg-amber-600 hover:bg-amber-700 focus:ring-amber-500',
      buttonDisabled: 'bg-amber-600/40 cursor-not-allowed',
      border: 'border-amber-500/30',
    },
  };

  const styles = variantStyles[variant];

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.15 }}
          className="fixed inset-0 z-50 flex items-center justify-center p-4"
        >
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={handleClose}
          />

          {/* Modal */}
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 10 }}
            transition={{ duration: 0.15 }}
            className={`relative w-full max-w-md bg-surface-raised border ${styles.border} rounded-xl shadow-2xl`}
          >
            {/* Header */}
            <div className="flex items-start gap-3 px-5 pt-5 pb-3">
              <div className={`mt-0.5 ${styles.icon}`}>
                <AlertTriangle className="w-5 h-5" />
              </div>
              <div className="flex-1">
                <h3 className="text-base font-semibold text-text-primary">{title}</h3>
                <p className="text-sm text-text-muted mt-1">{message}</p>
              </div>
              <button onClick={handleClose} className="btn btn-ghost p-1">
                <X className="w-4 h-4" />
              </button>
            </div>

            {/* Confirmation input */}
            <div className="px-5 pb-4">
              <label className="block text-xs text-text-muted mb-1.5">
                Type <span className="font-mono font-semibold text-text-secondary">{confirmText}</span> to confirm
              </label>
              <input
                type="text"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                placeholder={confirmText}
                className="w-full px-3 py-2 bg-surface-overlay border border-border-default rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                autoFocus
              />
            </div>

            {/* Actions */}
            <div className="flex items-center justify-end gap-2 px-5 pb-5">
              <button onClick={handleClose} className="btn btn-secondary">
                Cancel
              </button>
              <button
                onClick={() => {
                  if (isMatch) {
                    onConfirm();
                    handleClose();
                  }
                }}
                disabled={!isMatch}
                className={`btn text-white transition-colors ${isMatch ? styles.button : styles.buttonDisabled}`}
              >
                Confirm
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

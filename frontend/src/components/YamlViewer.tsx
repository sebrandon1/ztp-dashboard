import { useState, useMemo } from 'react';
import { motion } from 'framer-motion';
import { Copy, Check, Search, X } from 'lucide-react';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';

interface YamlViewerProps {
  data: Record<string, unknown>;
  title?: string;
  onClose: () => void;
}

export default function YamlViewer({ data, title, onClose }: YamlViewerProps) {
  const [copied, setCopied] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');

  const content = useMemo(() => JSON.stringify(data, null, 2), [data]);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard API may not be available
    }
  };

  const highlightedLines = useMemo(() => {
    if (!searchTerm) return new Set<number>();
    const lines = content.split('\n');
    const matches = new Set<number>();
    const term = searchTerm.toLowerCase();
    lines.forEach((line, i) => {
      if (line.toLowerCase().includes(term)) {
        matches.add(i + 1);
      }
    });
    return matches;
  }, [content, searchTerm]);

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      className="fixed inset-0 z-50 flex justify-end"
      onClick={onClose}
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" />

      {/* Panel */}
      <motion.div
        initial={{ x: '100%' }}
        animate={{ x: 0 }}
        exit={{ x: '100%' }}
        transition={{ type: 'spring', damping: 25, stiffness: 300 }}
        className="relative w-full max-w-2xl bg-surface-raised border-l border-border-subtle flex flex-col h-full"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border-subtle shrink-0">
          <h3 className="text-sm font-semibold text-text-primary truncate">
            {title || 'Resource Viewer'}
          </h3>
          <div className="flex items-center gap-2">
            <button
              onClick={handleCopy}
              className="btn btn-ghost p-1.5 gap-1 text-xs"
              title="Copy to clipboard"
            >
              {copied ? (
                <Check className="w-3.5 h-3.5 text-emerald-400" />
              ) : (
                <Copy className="w-3.5 h-3.5" />
              )}
              {copied ? 'Copied' : 'Copy'}
            </button>
            <button onClick={onClose} className="btn btn-ghost p-1.5">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Search */}
        <div className="px-4 py-2 border-b border-border-subtle shrink-0">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-text-muted" />
            <input
              type="text"
              placeholder="Search content..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-8 pr-4 py-1.5 bg-surface-overlay border border-border-default rounded-md text-xs text-text-primary placeholder:text-text-muted focus:outline-none focus:ring-1 focus:ring-primary-500"
            />
            {searchTerm && (
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-text-muted">
                {highlightedLines.size} match{highlightedLines.size !== 1 ? 'es' : ''}
              </span>
            )}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto">
          <SyntaxHighlighter
            language="json"
            style={oneDark}
            showLineNumbers
            wrapLongLines
            lineProps={(lineNumber: number) => ({
              style: highlightedLines.has(lineNumber)
                ? { backgroundColor: 'rgba(250, 204, 21, 0.15)' }
                : {},
            })}
            customStyle={{
              margin: 0,
              borderRadius: 0,
              background: 'transparent',
              fontSize: '0.75rem',
            }}
          >
            {content}
          </SyntaxHighlighter>
        </div>
      </motion.div>
    </motion.div>
  );
}

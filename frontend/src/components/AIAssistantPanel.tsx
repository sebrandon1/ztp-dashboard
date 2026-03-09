import { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Sparkles, Send, RotateCcw, Bot } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import { useAI } from '../hooks/useAI';
import type { AIModel } from '../types/api';
import { aiAPI } from '../lib/api';

interface AIAssistantPanelProps {
  initialContext?: Record<string, unknown>;
  diagnoseType?: string;
}

export default function AIAssistantPanel({ initialContext, diagnoseType }: AIAssistantPanelProps) {
  const { response, isStreaming, error, diagnose, reset } = useAI();
  const [models, setModels] = useState<AIModel[]>([]);
  const [selectedModel, setSelectedModel] = useState<string>('');
  const [aiConnected, setAiConnected] = useState(false);

  useEffect(() => {
    aiAPI.getStatus()
      .then((status) => {
        setAiConnected(status.connected);
        setSelectedModel(status.defaultModel);
      })
      .catch(() => setAiConnected(false));

    aiAPI.getModels()
      .then(setModels)
      .catch(() => {});
  }, []);

  const handleDiagnose = () => {
    const context = initialContext || {};
    diagnose(context, selectedModel, diagnoseType);
  };

  if (!aiConnected) {
    return (
      <div className="card p-6 text-center">
        <Bot className="w-12 h-12 text-text-muted mx-auto mb-3" />
        <h3 className="text-sm font-semibold text-text-primary mb-1">AI Diagnostics Unavailable</h3>
        <p className="text-xs text-text-muted">
          Connect to an ollama instance in Settings to enable AI-powered diagnostics.
        </p>
      </div>
    );
  }

  return (
    <div className="card overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-border-subtle bg-primary-600/5">
        <div className="flex items-center gap-2">
          <Sparkles className="w-4 h-4 text-primary-400" />
          <h3 className="text-sm font-semibold text-text-primary">AI Diagnostics</h3>
        </div>
        <div className="flex items-center gap-2">
          {models.length > 0 && (
            <select
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
              className="text-xs bg-surface-raised border border-border-default rounded-lg px-2 py-1 text-text-secondary focus:outline-none focus:ring-1 focus:ring-primary-500"
            >
              {models.map((m) => (
                <option key={m.name} value={m.name}>{m.name}</option>
              ))}
            </select>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="p-4">
        {!response && !isStreaming && !error && (
          <div className="text-center py-8">
            <Bot className="w-10 h-10 text-text-muted mx-auto mb-3 opacity-50" />
            <p className="text-sm text-text-muted mb-4">
              Click analyze to get AI-powered diagnostics for this context.
            </p>
            <button onClick={handleDiagnose} className="btn btn-primary gap-2">
              <Send className="w-4 h-4" />
              Analyze
            </button>
          </div>
        )}

        {isStreaming && !response && (
          <div className="flex items-center gap-3 py-8 justify-center">
            <ThinkingDots />
            <span className="text-sm text-text-secondary">Thinking...</span>
          </div>
        )}

        {(response || (isStreaming && response)) && (
          <div className="prose prose-invert prose-sm max-w-none">
            <ReactMarkdown>{response}</ReactMarkdown>
            {isStreaming && <span className="inline-block w-2 h-4 bg-primary-400 animate-pulse ml-0.5" />}
          </div>
        )}

        {error && (
          <div className="text-center py-6">
            <p className="text-sm text-red-400 mb-3">{error}</p>
            <button onClick={handleDiagnose} className="btn btn-secondary gap-2">
              <RotateCcw className="w-4 h-4" />
              Retry
            </button>
          </div>
        )}

        {response && !isStreaming && (
          <div className="mt-4 pt-4 border-t border-border-subtle flex gap-2">
            <button onClick={handleDiagnose} className="btn btn-secondary gap-2 text-xs">
              <RotateCcw className="w-3 h-3" />
              Re-analyze
            </button>
            <button onClick={reset} className="btn btn-ghost text-xs">
              Clear
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function ThinkingDots() {
  return (
    <div className="flex gap-1">
      {[0, 1, 2].map((i) => (
        <motion.div
          key={i}
          className="w-2 h-2 rounded-full bg-primary-400"
          animate={{ opacity: [0.3, 1, 0.3] }}
          transition={{ duration: 1.2, repeat: Infinity, delay: i * 0.2 }}
        />
      ))}
    </div>
  );
}

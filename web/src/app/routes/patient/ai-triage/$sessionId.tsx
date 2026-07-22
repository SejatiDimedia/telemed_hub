import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useAISession, useSendMessage } from '@/features/ai-assistant/hooks/use-ai';
import { ChatBubble } from '@/features/ai-assistant/components/ChatBubble';
import { TriageResultCard } from '@/features/ai-assistant/components/TriageResultCard';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { useState, useRef, useEffect } from 'react';
import type { AISuggestion } from '@/features/ai-assistant/types';

export const Route = createFileRoute('/patient/ai-triage/$sessionId')({
  component: AISessionChat,
});

function AISessionChat() {
  const { sessionId } = Route.useParams();
  const navigate = useNavigate();
  const { data: sessionResponse, isLoading, error } = useAISession(sessionId);
  const sendMessage = useSendMessage();

  const [input, setInput] = useState('');
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const session = sessionResponse;
  const isClosed = session?.status === 'closed';

  // Auto-scroll to bottom
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [session?.suggestions, sendMessage.isPending]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || sendMessage.isPending || isClosed) return;

    sendMessage.mutate(
      { id: sessionId, req: { message: input } },
      {
        onSuccess: () => {
          setInput('');
        },
      }
    );
  };

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-[50vh]">
        <span className="material-symbols-outlined animate-spin text-[32px] text-primary">progress_activity</span>
      </div>
    );
  }

  if (error || !session) {
    return (
      <div className="text-center p-8 border border-error-container/50 bg-error-container/10 rounded-2xl max-w-lg mx-auto mt-10">
        <span className="material-symbols-outlined text-error text-[40px] mb-4">error</span>
        <h3 className="text-lg font-bold text-on-surface mb-2">Gagal memuat sesi</h3>
        <p className="text-on-surface-variant mb-6">Terjadi kesalahan saat mengambil data sesi Triage.</p>
        <Button onClick={() => navigate({ to: '/patient/ai-triage' })}>
          Kembali ke Riwayat
        </Button>
      </div>
    );
  }

  return (
    <div className="flex flex-col h-[calc(100vh-8rem)] max-w-4xl mx-auto border border-outline-variant/30 rounded-2xl overflow-hidden bg-surface-container-lowest shadow-level-1">
      {/* Header */}
      <div className="flex items-center gap-4 p-4 border-b border-outline-variant/20 bg-surface">
        <Button
          variant="text"
          onClick={() => navigate({ to: '/patient/ai-triage' })}
          className="shrink-0 w-10 h-10 p-0 rounded-full text-on-surface-variant hover:bg-surface-variant/50"
        >
          <span className="material-symbols-outlined text-[24px]">arrow_back</span>
        </Button>
        <div className="flex-1">
          <h2 className="text-title-lg font-bold text-on-surface flex items-center gap-3">
            Konsultasi AI Triage
            {isClosed && (
              <span className="text-label-sm font-medium bg-surface-variant text-on-surface-variant px-2.5 py-1 rounded-md">
                Selesai
              </span>
            )}
          </h2>
          <p className="text-body-sm text-on-surface-variant mt-0.5">Sesi ID: <span className="font-mono text-xs opacity-80">{session.id.split('-')[0]}</span></p>
        </div>
      </div>

      {/* Chat Area */}
      <div className="flex-1 overflow-y-auto p-4 md:p-6 space-y-6 bg-surface-container-lowest/50 relative">
        <ChatBubble
          role="assistant"
          content="Halo! Saya adalah AI Triage Assistant. Ceritakan keluhan atau gejala yang Anda alami secara detail, dan saya akan membantu menilai tingkat urgensi dan spesialisasi dokter yang tepat untuk Anda."
          timestamp={session.created_at}
        />

        {session.suggestions?.map((sug: AISuggestion) => (
          <div key={sug.id} className="space-y-6">
            <ChatBubble
              role="user"
              content={sug.input_summary}
              timestamp={sug.created_at}
            />
            
            <div className="flex flex-col gap-2">
              <ChatBubble
                role="assistant"
                content="Berdasarkan keluhan yang Anda sampaikan, berikut adalah hasil analisis saya:"
                timestamp={sug.created_at}
              />
              <div className="pl-14 pr-4">
                <TriageResultCard
                  urgency={sug.suggested_urgency}
                  specialty={sug.suggested_specialty}
                  disclaimer={
                    sug.disclaimer_shown
                      ? 'Ini adalah saran awal berbasis AI dan bukan merupakan diagnosis medis. Konsultasikan dengan dokter untuk memastikannya.'
                      : 'Bukan pengganti diagnosis medis.'
                  }
                />
              </div>
            </div>
          </div>
        ))}

        {sendMessage.isPending && (
          <div className="flex justify-end mb-4">
            <div className="flex items-center gap-2 bg-primary-container/20 text-primary px-5 py-3 rounded-3xl rounded-br-sm shadow-sm">
              <span className="material-symbols-outlined animate-spin text-[18px]">progress_activity</span>
              <span className="text-body-sm font-medium">Sedang menganalisis...</span>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} className="h-4" />
      </div>

      {/* Input Area */}
      <div className="p-4 md:p-6 bg-surface border-t border-outline-variant/20">
        {isClosed ? (
          <div className="bg-surface-variant/30 border border-outline-variant/40 rounded-xl p-4 text-center">
            <span className="material-symbols-outlined text-on-surface-variant text-[24px] mb-2">lock</span>
            <p className="text-body-md text-on-surface-variant font-medium">Sesi ini sudah ditutup</p>
            <p className="text-body-sm text-on-surface-variant/80 mt-1">Silakan mulai sesi baru dari halaman sebelumnya jika ada keluhan lain.</p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="flex items-end gap-3">
            <div className="flex-1">
              <Input
                placeholder="Deskripsikan gejala Anda (minimal 5 karakter)..."
                value={input}
                onChange={(e: any) => setInput(e.target.value)}
                disabled={sendMessage.isPending}
                className="w-full bg-surface-container-lowest border-outline-variant focus:border-primary focus:ring-primary shadow-sm rounded-xl py-3 px-4"
                autoComplete="off"
              />
            </div>
            <Button
              type="submit"
              disabled={sendMessage.isPending || input.trim().length < 5}
              className="shrink-0 h-[48px] px-6 rounded-xl shadow-sm"
              leftIcon={sendMessage.isPending ? 'progress_activity' : 'send'}
            >
              Kirim
            </Button>
          </form>
        )}
      </div>
    </div>
  );
}

import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { useAISessions, useCreateAISession } from '@/features/ai-assistant/hooks/use-ai';
import { Badge } from '@/components/ui/Badge';
import { useToastStore } from '@/stores/toast-store';

export const Route = createFileRoute('/patient/ai-triage/')({
  component: AITriageIndex,
});

function AITriageIndex() {
  const { data: sessionsResponse, isLoading } = useAISessions();
  const createSession = useCreateAISession();
  const navigate = useNavigate();
  const addToast = useToastStore((state: any) => state.addToast);

  // apiClient unwraps the { data: ... } envelope, so sessionsResponse is already the array
  const sessions = Array.isArray(sessionsResponse) ? sessionsResponse : [];
  const activeSession = sessions.find((s) => s.status === 'active');

  const handleStartSession = () => {
    if (activeSession) {
      navigate({ to: `/patient/ai-triage/${activeSession.id}` });
      return;
    }
    
    createSession.mutate(undefined, {
      onSuccess: (res: any) => {
        navigate({ to: `/patient/ai-triage/${res.id}` });
      },
      onError: (error: any) => {
        addToast({
          type: 'error',
          title: 'Gagal Memulai Sesi',
          message: error.message || 'Terjadi kesalahan saat memulai sesi.',
        });
      }
    });
  };

  return (
    <div className="space-y-6 max-w-4xl mx-auto pb-10">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-foreground flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[24px]">smart_toy</span>
          AI Triage Assistant
        </h1>
        <p className="text-muted-foreground mt-1">
          Konsultasikan gejala awal Anda secara mandiri dengan asisten cerdas kami sebelum membuat janji temu.
        </p>
      </div>

      <Card className="border-primary/20 bg-primary/5 shadow-sm">
        <CardHeader>
          <CardTitle>Mulai Deteksi Dini</CardTitle>
          <CardDescription>
            Sistem AI akan menganalisis gejala Anda dan memberikan rekomendasi tingkat urgensi serta spesialisasi dokter yang tepat.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button
            onClick={handleStartSession}
            disabled={createSession.isPending}
            className="w-full sm:w-auto"
            leftIcon={createSession.isPending ? 'progress_activity' : activeSession ? 'chat' : 'add'}
          >
            {activeSession ? 'Lanjutkan Sesi Aktif' : 'Mulai Sesi Baru'}
          </Button>
        </CardContent>
      </Card>

      <div className="space-y-4">
        <h2 className="text-lg font-semibold tracking-tight">Riwayat Triage</h2>
        {isLoading ? (
          <div className="flex justify-center p-8">
            <span className="material-symbols-outlined text-primary animate-spin text-[24px]">progress_activity</span>
          </div>
        ) : sessions.length === 0 ? (
          <div className="text-center p-8 border border-dashed rounded-lg text-muted-foreground">
            Belum ada riwayat sesi triage.
          </div>
        ) : (
          <div className="grid gap-4">
            {sessions.map((session) => (
              <Card
                key={session.id}
                className="cursor-pointer hover:border-primary/50 transition-colors"
                onClick={() => navigate({ to: `/patient/ai-triage/${session.id}` })}
              >
                <CardContent className="p-4 flex items-center justify-between">
                  <div className="flex flex-col gap-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">Sesi Triage</span>
                      <Badge variant={session.status === 'active' ? 'primary' : 'neutral'}>
                        {session.status}
                      </Badge>
                    </div>
                    <div className="flex items-center gap-1 text-sm text-muted-foreground">
                      <span className="material-symbols-outlined text-[14px]">schedule</span>
                      {new Date(session.created_at).toLocaleString()}
                    </div>
                  </div>
                  <span className="material-symbols-outlined text-muted-foreground text-[20px]">chat</span>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

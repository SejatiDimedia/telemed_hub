
import { Card, CardContent, CardHeader, CardTitle, CardFooter } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Link } from '@tanstack/react-router';

interface TriageResultCardProps {
  urgency: 'low' | 'medium' | 'high';
  specialty: string;
  disclaimer: string;
}

export function TriageResultCard({ urgency, specialty, disclaimer }: TriageResultCardProps) {
  const urgencyConfig = {
    low: { color: 'bg-green-100 text-green-800 border-green-200', label: 'Rendah' },
    medium: { color: 'bg-yellow-100 text-yellow-800 border-yellow-200', label: 'Sedang' },
    high: { color: 'bg-red-100 text-red-800 border-red-200', label: 'Tinggi' },
  };

  const config = urgencyConfig[urgency] || urgencyConfig.medium;

  return (
    <Card className="w-full max-w-sm mt-2 border-primary/20 shadow-md">
      <CardHeader className="pb-3 bg-muted/30">
        <div className="flex items-center justify-between">
          <CardTitle className="text-base font-semibold flex items-center gap-2">
            <span className="material-symbols-outlined text-primary text-[20px]">stethoscope</span>
            Hasil Triage AI
          </CardTitle>
          <Badge variant="neutral" className={config.color}>
            Urgensi: {config.label}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="pt-4 pb-3 space-y-4">
        <div>
          <p className="text-xs text-muted-foreground uppercase tracking-wider font-semibold mb-1">
            Rekomendasi Spesialisasi
          </p>
          <p className="font-medium text-foreground capitalize flex items-center gap-2">
            {specialty.replace(/_/g, ' ')}
          </p>
        </div>
        
        <div className="bg-amber-50 border border-amber-200 rounded-md p-3 flex gap-2 items-start text-amber-800">
          <span className="material-symbols-outlined text-[16px] shrink-0 mt-0.5">warning</span>
          <p className="text-xs leading-relaxed">
            <span className="font-semibold block mb-1">Medical Disclaimer</span>
            {disclaimer}
          </p>
        </div>
      </CardContent>
      <CardFooter className="pt-0 pb-4">
        <Link to="/patient/appointments" search={{ specialty }} className="w-full">
          <Button className="w-full" size="sm">
            Cari Dokter {specialty.replace(/_/g, ' ')}
          </Button>
        </Link>
      </CardFooter>
    </Card>
  );
}

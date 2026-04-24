import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface QueryErrorCardProps {
  title: string;
  message: string;
  hint?: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
}

export function QueryErrorCard({
  title,
  message,
  hint,
  onRetry,
  retryLabel = "Retry",
  className
}: QueryErrorCardProps) {
  return (
    <Card className={cn("border-rose-200 bg-rose-50", className)}>
      <CardHeader>
        <CardTitle className="text-rose-700">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-rose-700">{message}</p>
        {hint ? <p className="text-sm text-rose-600/90">{hint}</p> : null}
        {onRetry ? (
          <div>
            <Button onClick={onRetry} size="sm">
              {retryLabel}
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

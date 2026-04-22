import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface FileEditorPanelProps {
  path: string;
  content: string;
  truncated: boolean;
  loading: boolean;
  onSave: (content: string) => Promise<void>;
}

export function FileEditorPanel({ path, content, truncated, loading, onSave }: FileEditorPanelProps) {
  const [draft, setDraft] = useState(content);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");

  useEffect(() => {
    setDraft(content);
    setMessage("");
  }, [content, path]);

  async function handleSave() {
    setMessage("");
    setSaving(true);
    try {
      await onSave(draft);
      setMessage("Saved successfully.");
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "Save failed");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Loading file...</CardTitle>
        </CardHeader>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between gap-4">
        <div>
          <CardTitle className="text-base">Editor</CardTitle>
          <p className="mt-1 text-xs text-slate-500">{path}</p>
        </div>
        <Button disabled={saving} onClick={handleSave} size="sm">
          {saving ? "Saving..." : "Save"}
        </Button>
      </CardHeader>
      <CardContent className="space-y-3">
        {truncated && (
          <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-700">
            File content was truncated by server-side max read size.
          </p>
        )}
        <textarea
          className="min-h-[320px] w-full rounded-md border border-slate-300 p-3 font-mono text-sm"
          onChange={(event) => setDraft(event.target.value)}
          value={draft}
        />
        {message && <p className="text-sm text-slate-600">{message}</p>}
      </CardContent>
    </Card>
  );
}

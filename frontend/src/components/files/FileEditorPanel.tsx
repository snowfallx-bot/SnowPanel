import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface FileEditorPanelProps {
  path: string;
  content: string;
  truncated: boolean;
  binary: boolean;
  loading: boolean;
  canDownload: boolean;
  downloading: boolean;
  downloadProgressText?: string;
  onDownload: () => void;
  onSave: (content: string) => Promise<void>;
}

export function FileEditorPanel({
  path,
  content,
  truncated,
  binary,
  loading,
  canDownload,
  downloading,
  downloadProgressText,
  onDownload,
  onSave
}: FileEditorPanelProps) {
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
        <div className="flex items-center gap-2">
          <Button disabled={!canDownload || downloading} onClick={onDownload} size="sm" variant="ghost">
            {downloading ? "Downloading..." : "Download"}
          </Button>
          <Button disabled={saving || binary || downloading} onClick={handleSave} size="sm">
            {saving ? "Saving..." : "Save"}
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        {downloading && downloadProgressText ? (
          <p className="rounded bg-sky-50 px-3 py-2 text-xs text-sky-700">{downloadProgressText}</p>
        ) : null}
        {binary && (
          <p className="rounded bg-slate-100 px-3 py-2 text-xs text-slate-700">
            This file appears to be binary or non UTF-8 text, so inline editing is disabled.
          </p>
        )}
        {truncated && (
          <p className="rounded bg-amber-50 px-3 py-2 text-xs text-amber-700">
            File content was truncated by server-side max read size.
          </p>
        )}
        {!binary && (
          <textarea
            className="min-h-[320px] w-full rounded-md border border-slate-300 p-3 font-mono text-sm"
            onChange={(event) => setDraft(event.target.value)}
            value={draft}
          />
        )}
        {message && <p className="text-sm text-slate-600">{message}</p>}
      </CardContent>
    </Card>
  );
}

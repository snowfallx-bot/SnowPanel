import { FormEvent, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface FilePathBarProps {
  path: string;
  onNavigate: (path: string) => void;
  onGoUp: () => void;
}

function normalizePath(path: string) {
  if (!path) {
    return "/";
  }
  return path.replace(/\\/g, "/");
}

export function FilePathBar({ path, onNavigate, onGoUp }: FilePathBarProps) {
  const normalized = normalizePath(path);
  const rawParts = normalized.split("/").filter(Boolean);
  const parts = ["/", ...rawParts];
  const [draftPath, setDraftPath] = useState(normalized);

  useEffect(() => {
    setDraftPath(normalized);
  }, [normalized]);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onNavigate(normalizePath(draftPath));
  }

  return (
    <div className="space-y-3 rounded-lg border border-slate-200 bg-white p-3">
      <form className="flex gap-2" onSubmit={handleSubmit}>
        <Input aria-label="Current path" value={draftPath} onChange={(event) => setDraftPath(event.target.value)} />
        <Button type="submit">Load</Button>
        <Button size="sm" variant="ghost" onClick={onGoUp} type="button">
          Up
        </Button>
      </form>
      <div className="flex flex-wrap items-center gap-1 text-sm text-slate-600">
        {parts.map((part, index) => {
          const target =
            index === 0
              ? "/"
              : `/${rawParts.slice(0, index).join("/")}`.replace("//", "/");

          return (
            <span className="flex items-center gap-1" key={`${part}-${index}`}>
              {index > 0 && <span>/</span>}
              <button className="rounded px-1 hover:bg-slate-100" onClick={() => onNavigate(target)} type="button">
                {part === "/" ? "root" : part}
              </button>
            </span>
          );
        })}
      </div>
    </div>
  );
}

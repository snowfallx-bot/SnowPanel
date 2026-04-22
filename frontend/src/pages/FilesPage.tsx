import { FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  createDirectory,
  deleteFile,
  listFiles,
  readTextFile,
  writeTextFile
} from "@/api/files";
import { FileEditorPanel } from "@/components/files/FileEditorPanel";
import { FilePathBar } from "@/components/files/FilePathBar";
import { FileTable } from "@/components/files/FileTable";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { FileEntry } from "@/types/file";

function parentPath(path: string) {
  if (!path || path === "/") {
    return "/";
  }
  const normalized = path.replace(/\\/g, "/").replace(/\/+$/, "");
  const index = normalized.lastIndexOf("/");
  if (index <= 0) {
    return "/";
  }
  return normalized.slice(0, index);
}

export function FilesPage() {
  const queryClient = useQueryClient();
  const [path, setPath] = useState("/tmp");
  const [mkdirName, setMkdirName] = useState("");
  const [selectedPath, setSelectedPath] = useState("");
  const [selectedContent, setSelectedContent] = useState("");
  const [selectedTruncated, setSelectedTruncated] = useState(false);

  const listQuery = useQuery({
    queryKey: ["files", path],
    queryFn: () => listFiles(path)
  });

  const readMutation = useMutation({
    mutationFn: readTextFile,
    onSuccess(data) {
      setSelectedPath(data.path);
      setSelectedContent(data.content);
      setSelectedTruncated(data.truncated);
    }
  });

  const writeMutation = useMutation({
    mutationFn: writeTextFile
  });

  const mkdirMutation = useMutation({
    mutationFn: createDirectory,
    onSuccess() {
      setMkdirName("");
      queryClient.invalidateQueries({ queryKey: ["files", path] });
    }
  });

  const deleteMutation = useMutation({
    mutationFn: deleteFile,
    onSuccess() {
      queryClient.invalidateQueries({ queryKey: ["files", path] });
      if (selectedPath) {
        setSelectedPath("");
        setSelectedContent("");
        setSelectedTruncated(false);
      }
    }
  });

  const message = useMemo(() => {
    if (listQuery.isError) {
      return listQuery.error instanceof Error ? listQuery.error.message : "Failed to list files";
    }
    if (readMutation.isError) {
      return readMutation.error instanceof Error ? readMutation.error.message : "Failed to read file";
    }
    if (mkdirMutation.isError) {
      return mkdirMutation.error instanceof Error ? mkdirMutation.error.message : "Failed to create directory";
    }
    if (deleteMutation.isError) {
      return deleteMutation.error instanceof Error ? deleteMutation.error.message : "Failed to delete path";
    }
    return "";
  }, [deleteMutation.error, deleteMutation.isError, listQuery.error, listQuery.isError, mkdirMutation.error, mkdirMutation.isError, readMutation.error, readMutation.isError]);

  async function handleOpen(entry: FileEntry) {
    if (entry.is_dir) {
      setPath(entry.path);
      return;
    }
    await readMutation.mutateAsync({
      path: entry.path,
      max_bytes: 1024 * 1024,
      encoding: "utf-8"
    });
  }

  async function handleSave(content: string) {
    if (!selectedPath) {
      throw new Error("No file selected");
    }

    await writeMutation.mutateAsync({
      path: selectedPath,
      content,
      create_if_not_exists: false,
      truncate: true,
      encoding: "utf-8"
    });
    setSelectedContent(content);
    queryClient.invalidateQueries({ queryKey: ["files", path] });
  }

  async function handleDelete(entry: FileEntry) {
    const ok = window.confirm(
      `Delete ${entry.is_dir ? "directory" : "file"} "${entry.path}"? This action cannot be undone.`
    );
    if (!ok) {
      return;
    }

    await deleteMutation.mutateAsync({
      path: entry.path,
      recursive: entry.is_dir
    });
  }

  async function handleCreateDirectory(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = mkdirName.trim();
    if (!name) {
      return;
    }
    const base = listQuery.data?.current_path || path;
    const nextPath = `${base.replace(/\/$/, "")}/${name}`.replace("//", "/");
    await mkdirMutation.mutateAsync({
      path: nextPath,
      create_parents: false
    });
  }

  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-2xl font-semibold text-slate-900">Files</h2>
        <p className="text-sm text-slate-500">Browse and edit text files through the Rust core-agent.</p>
      </div>

      <FilePathBar
        onGoUp={() => setPath(parentPath(listQuery.data?.current_path || path))}
        onNavigate={(target) => setPath(target)}
        path={listQuery.data?.current_path || path}
      />

      <div className="grid gap-4 xl:grid-cols-[2fr_1fr]">
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Create Directory</CardTitle>
            </CardHeader>
            <CardContent>
              <form className="flex gap-2" onSubmit={handleCreateDirectory}>
                <Input onChange={(event) => setMkdirName(event.target.value)} placeholder="new-folder-name" value={mkdirName} />
                <Button disabled={mkdirMutation.isPending} type="submit">
                  {mkdirMutation.isPending ? "Creating..." : "Create"}
                </Button>
              </form>
            </CardContent>
          </Card>

          {listQuery.isLoading ? (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Loading directory...</CardTitle>
              </CardHeader>
            </Card>
          ) : (
            <FileTable entries={listQuery.data?.entries || []} onDelete={handleDelete} onOpen={handleOpen} />
          )}
        </div>

        <FileEditorPanel
          content={selectedContent}
          loading={readMutation.isPending}
          onSave={handleSave}
          path={selectedPath || "No file selected"}
          truncated={selectedTruncated}
        />
      </div>

      {message && (
        <p className="rounded-md border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700">{message}</p>
      )}
    </div>
  );
}

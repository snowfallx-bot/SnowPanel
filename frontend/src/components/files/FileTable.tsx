import { FileEntry } from "@/types/file";
import { Button } from "@/components/ui/button";

interface FileTableProps {
  entries: FileEntry[];
  onOpen: (entry: FileEntry) => void;
  onRename: (entry: FileEntry) => void;
  onDelete: (entry: FileEntry) => void;
}

function formatTime(unix: number) {
  if (!unix) {
    return "-";
  }
  return new Date(unix * 1000).toLocaleString();
}

function formatSize(size: number, isDir: boolean) {
  if (isDir) {
    return "-";
  }
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}

export function FileTable({ entries, onOpen, onRename, onDelete }: FileTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border border-slate-200 bg-white">
      <table className="w-full text-left text-sm">
        <thead className="bg-slate-50 text-slate-600">
          <tr>
            <th className="px-4 py-3">Name</th>
            <th className="px-4 py-3">Type</th>
            <th className="px-4 py-3">Size</th>
            <th className="px-4 py-3">Modified</th>
            <th className="px-4 py-3">Action</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr className="border-t border-slate-200" key={entry.path}>
              <td className="px-4 py-3">
                <button className="font-medium text-panel-700 hover:underline" onClick={() => onOpen(entry)} type="button">
                  {entry.name}
                </button>
              </td>
              <td className="px-4 py-3">{entry.is_dir ? "Directory" : "File"}</td>
              <td className="px-4 py-3">{formatSize(entry.size, entry.is_dir)}</td>
              <td className="px-4 py-3">{formatTime(entry.modified_at_unix)}</td>
              <td className="px-4 py-3">
                <div className="flex gap-2">
                  <Button size="sm" variant="ghost" onClick={() => onRename(entry)}>
                    Rename
                  </Button>
                  <Button size="sm" variant="ghost" onClick={() => onDelete(entry)}>
                    Delete
                  </Button>
                </div>
              </td>
            </tr>
          ))}
          {entries.length === 0 && (
            <tr>
              <td className="px-4 py-8 text-center text-slate-500" colSpan={5}>
                Directory is empty.
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

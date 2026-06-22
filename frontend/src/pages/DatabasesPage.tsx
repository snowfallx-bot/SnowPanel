import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { listDatabaseInstances, createDatabaseInstance, updateDatabaseInstance, deleteDatabaseInstance, testConnection } from "@/api/database";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { QueryErrorCard } from "@/components/ui/query-error-card";
import { describeApiError } from "@/lib/http";
import { hostScopeKey, useHostStore } from "@/store/host-store";
import { DatabaseInstance, CreateDatabaseInstanceRequest, UpdateDatabaseInstanceRequest } from "@/types/database";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";

const ENGINE_OPTIONS: Array<{ value: string; label: string }> = [
  { value: "postgresql", label: "PostgreSQL" },
  { value: "mysql", label: "MySQL" },
];

export function DatabasesPage() {
  const queryClient = useQueryClient();
  const selectedHostId = useHostStore((state) => state.selectedHostId);
  const hostScope = hostScopeKey(selectedHostId);
  const [editingInstance, setEditingInstance] = useState<DatabaseInstance | null>(null);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  const instancesQuery = useQuery({
    queryKey: ["database-instances", hostScope],
    queryFn: listDatabaseInstances,
  });

  const createMutation = useMutation({
    mutationFn: createDatabaseInstance,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["database-instances", hostScope] });
      setIsCreateDialogOpen(false);
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateDatabaseInstanceRequest }) => updateDatabaseInstance(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["database-instances", hostScope] });
      setIsEditDialogOpen(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => deleteDatabaseInstance(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["database-instances", hostScope] });
      setIsDeleteDialogOpen(false);
    },
  });

  const testMutation = useMutation({
    mutationFn: testConnection,
    onSuccess: (data) => {
      setTestResult({ success: data.success, message: data.message });
    },
  });

  const loadError = instancesQuery.isError
    ? describeApiError(instancesQuery.error, "Failed to load database instances.")
    : null;

  function handleCreate() {
    setEditingInstance({
      id: 0,
      name: "",
      engine: "postgresql",
      host: "",
      port: 5432,
      username: "",
      status: 1,
      created_at: "",
      updated_at: "",
    });
    setIsCreateDialogOpen(true);
    setTestResult(null);
  }

  function handleEdit(instance: DatabaseInstance) {
    setEditingInstance({ ...instance });
    setIsEditDialogOpen(true);
    setTestResult(null);
  }

  function handleDelete(instance: DatabaseInstance) {
    setEditingInstance({ ...instance });
    setIsDeleteDialogOpen(true);
  }

  function handleTestConnection() {
    if (!editingInstance) return;
    testMutation.mutate({
      name: editingInstance.name,
      engine: editingInstance.engine,
      host: editingInstance.host,
      port: editingInstance.port,
      username: editingInstance.username,
      password: "****",
    });
  }

  function handleCreateSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!editingInstance) return;
    createMutation.mutate({
      name: editingInstance.name,
      engine: editingInstance.engine,
      host: editingInstance.host,
      port: editingInstance.port,
      username: editingInstance.username,
      password: "****",
    });
  }

  function handleUpdateSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!editingInstance) return;
    updateMutation.mutate({
      id: editingInstance.id,
      data: {
        name: editingInstance.name,
        engine: editingInstance.engine,
        host: editingInstance.host,
        port: editingInstance.port,
        username: editingInstance.username,
        status: editingInstance.status,
      },
    });
  }

  return (
    <div className="space-y-4">
      {loadError && <QueryErrorCard error={loadError} />}
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">Database Instances</h2>
        <Button onClick={handleCreate}>Add Instance</Button>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {instancesQuery.data?.items.map((instance) => (
          <Card key={instance.id} className="overflow-hidden">
            <CardHeader className="bg-slate-50">
              <CardTitle className="text-lg">{instance.name}</CardTitle>
              <p className="text-xs text-slate-500">{instance.engine} - {instance.host}:{instance.port}</p>
            </CardHeader>
            <CardContent className="p-4">
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-slate-500">Username</span>
                  <span className="font-medium">{instance.username}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Status</span>
                  <span className={`font-medium ${instance.status === 1 ? "text-emerald-600" : "text-rose-600"}`}>
                    {instance.status === 1 ? "Enabled" : "Disabled"}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Databases</span>
                  <span className="font-medium">{instance.databases_count ?? 0}</span>
                </div>
              </div>
              <div className="mt-4 flex gap-2">
                <Button variant="secondary" size="sm" onClick={() => handleEdit(instance)}>
                  Edit
                </Button>
                <Button variant="destructive" size="sm" onClick={() => handleDelete(instance)}>
                  Delete
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Create Dialog */}
      <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add Database Instance</DialogTitle>
            <DialogDescription>Configure a new database instance for management.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleCreateSubmit} className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="name">Name</Label>
              <Input
                id="name"
                value={editingInstance?.name || ""}
                onChange={(e) => setEditingInstance({ ...editingInstance!, name: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="engine">Engine</Label>
              <Select
                value={editingInstance?.engine || "postgresql"}
                onValueChange={(value) => setEditingInstance({ ...editingInstance!, engine: value as string })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select engine" />
                </SelectTrigger>
                <SelectContent>
                  {ENGINE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="host">Host</Label>
              <Input
                id="host"
                value={editingInstance?.host || ""}
                onChange={(e) => setEditingInstance({ ...editingInstance!, host: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="port">Port</Label>
              <Input
                id="port"
                type="number"
                value={editingInstance?.port || 5432}
                onChange={(e) => setEditingInstance({ ...editingInstance!, port: parseInt(e.target.value, 10) })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="username">Username</Label>
              <Input
                id="username"
                value={editingInstance?.username || ""}
                onChange={(e) => setEditingInstance({ ...editingInstance!, username: e.target.value })}
                required
              />
            </div>
            {testResult && (
              <div className={`rounded-md p-3 text-sm ${testResult.success ? "bg-emerald-50 text-emerald-700" : "bg-rose-50 text-rose-700"}`}>
                {testResult.message}
              </div>
            )}
            <DialogFooter>
              <Button type="button" variant="secondary" onClick={handleTestConnection}>
                Test Connection
              </Button>
              <Button type="submit">Create</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Edit Database Instance</DialogTitle>
            <DialogDescription>Modify the database instance configuration.</DialogDescription>
          </DialogHeader>
          <form onSubmit={handleUpdateSubmit} className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="edit-name">Name</Label>
              <Input
                id="edit-name"
                value={editingInstance?.name || ""}
                onChange={(e) => setEditingInstance({ ...editingInstance!, name: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="edit-engine">Engine</Label>
              <Select
                value={editingInstance?.engine || "postgresql"}
                onValueChange={(value) => setEditingInstance({ ...editingInstance!, engine: value as string })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select engine" />
                </SelectTrigger>
                <SelectContent>
                  {ENGINE_OPTIONS.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="edit-host">Host</Label>
              <Input
                id="edit-host"
                value={editingInstance?.host || ""}
                onChange={(e) => setEditingInstance({ ...editingInstance!, host: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="edit-port">Port</Label>
              <Input
                id="edit-port"
                type="number"
                value={editingInstance?.port || 5432}
                onChange={(e) => setEditingInstance({ ...editingInstance!, port: parseInt(e.target.value, 10) })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="edit-username">Username</Label>
              <Input
                id="edit-username"
                value={editingInstance?.username || ""}
                onChange={(e) => setEditingInstance({ ...editingInstance!, username: e.target.value })}
                required
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="edit-status">Status</Label>
              <Select
                value={editingInstance?.status?.toString() || "1"}
                onValueChange={(value) => setEditingInstance({ ...editingInstance!, status: parseInt(value, 10) })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1">Enabled</SelectItem>
                  <SelectItem value="0">Disabled</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <DialogFooter>
              <Button type="submit">Save</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete Database Instance</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this database instance? This will also delete all associated databases.
            </DialogDescription>
          </DialogHeader>
          <div className="text-sm text-slate-600">
            Instance: <span className="font-medium">{editingInstance?.name}</span>
          </div>
          <DialogFooter>
            <Button variant="secondary" onClick={() => setIsDeleteDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={() => editingInstance && deleteMutation.mutate(editingInstance.id)}>
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

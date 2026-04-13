import { useState, useMemo, useRef, useCallback } from "react"
import { useTranslation } from "react-i18next"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import { toast } from "sonner"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from "@/components/ui/sheet"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  DataTableActionsCell,
  DataTableActionsHead,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
} from "@/components/ui/data-table"
import {
  Search,
  Plus,
  Star,
  Users,
  ChevronRight,
  MoreHorizontal,
  ArrowRightLeft,
  Building2,
  Trash2,
  FolderOpen,
  ChevronsUpDown,
  Check,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { ChangePositionSheet } from "../../components/change-position-sheet"
import { UserOrgSheet } from "../../components/user-org-sheet"

interface TreeNode {
  id: number
  name: string
  code: string
  memberCount: number
  children?: TreeNode[]
}

interface MemberItem {
  userId: number
  username: string
  email: string
  avatar: string
  departmentId: number
  positionId: number
  isPrimary: boolean
  assignmentId: number
  createdAt: string
}

interface PositionItem {
  id: number
  name: string
  code: string
  isActive: boolean
}

interface UserItem {
  id: number
  username: string
  email: string
  avatar: string
}

function collectAllIds(nodes: TreeNode[]): number[] {
  const ids: number[] = []
  for (const n of nodes) {
    ids.push(n.id)
    if (n.children) ids.push(...collectAllIds(n.children))
  }
  return ids
}

function collectExpandedIds(nodes: TreeNode[], maxDepth: number, depth = 0): number[] {
  const ids: number[] = []
  for (const node of nodes) {
    if (depth < maxDepth && node.children?.length) {
      ids.push(node.id)
      ids.push(...collectExpandedIds(node.children, maxDepth, depth + 1))
    }
  }
  return ids
}

function findNodeById(nodes: TreeNode[], id: number): TreeNode | null {
  for (const node of nodes) {
    if (node.id === id) return node
    if (node.children) {
      const result = findNodeById(node.children, id)
      if (result) return result
    }
  }
  return null
}

function filterTree(nodes: TreeNode[], keyword: string): TreeNode[] {
  if (!keyword) return nodes
  const lower = keyword.toLowerCase()
  const result: TreeNode[] = []
  for (const node of nodes) {
    const childMatches = node.children ? filterTree(node.children, keyword) : []
    const selfMatches =
      node.name.toLowerCase().includes(lower) || node.code.toLowerCase().includes(lower)
    if (selfMatches || childMatches.length > 0) {
      result.push({ ...node, children: selfMatches ? node.children : childMatches })
    }
  }
  return result
}

function DepartmentTreeItem({
  node,
  selectedId,
  onSelect,
  expanded,
  onToggleExpand,
  depth,
}: {
  node: TreeNode
  selectedId: number | null
  onSelect: (id: number) => void
  expanded: Set<number>
  onToggleExpand: (id: number) => void
  depth: number
}) {
  const hasChildren = node.children && node.children.length > 0
  const isExpanded = expanded.has(node.id)
  const isSelected = selectedId === node.id
  return (
    <div className="space-y-1">
      <button
        type="button"
        onClick={() => onSelect(node.id)}
        className={cn(
          "group flex w-full min-w-0 items-center gap-2 rounded-lg px-2.5 py-2 text-left text-sm transition-colors duration-150",
          isSelected
            ? "bg-accent text-accent-foreground"
            : "text-foreground/88 hover:bg-accent/60"
        )}
        style={{ paddingLeft: `${depth * 14 + 12}px` }}
      >
        {hasChildren ? (
          <span
            className={cn(
              "flex h-5 w-5 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors",
              isSelected ? "text-foreground/80" : "group-hover:text-foreground"
            )}
            onClick={(e) => {
              e.stopPropagation()
              onToggleExpand(node.id)
            }}
          >
            <ChevronRight
              className={cn(
                "h-3.5 w-3.5 transition-transform duration-200",
                isExpanded && "rotate-90"
              )}
            />
          </span>
        ) : (
          <span className="w-5 shrink-0" />
        )}
        <span className="truncate flex-1 font-medium">{node.name}</span>
        {node.memberCount > 0 && (
          <span
            className={cn(
              "shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium tabular-nums",
              isSelected
                ? "bg-background text-foreground"
                : "bg-muted text-muted-foreground"
            )}
          >
            {node.memberCount}
          </span>
        )}
      </button>
      {hasChildren && isExpanded && (
        <div>
          {node.children!.map((child) => (
            <DepartmentTreeItem
              key={child.id}
              node={child}
              selectedId={selectedId}
              onSelect={onSelect}
              expanded={expanded}
              onToggleExpand={onToggleExpand}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export function Component() {
  const { t } = useTranslation(["org", "common"])
  const queryClient = useQueryClient()
  const [selectedDeptId, setSelectedDeptId] = useState<number | null>(null)
  const [expanded, setExpanded] = useState<Set<number>>(new Set())
  const [deptSearch, setDeptSearch] = useState("")
  const [sheetOpen, setSheetOpen] = useState(false)
  const [selectedUserId, setSelectedUserId] = useState<string>("")
  const [selectedUserObj, setSelectedUserObj] = useState<UserItem | null>(null)
  const [userComboOpen, setUserComboOpen] = useState(false)
  const [selectedPositionId, setSelectedPositionId] = useState<string>("")
  const [isPrimary, setIsPrimary] = useState(false)
  const [userKeyword, setUserKeyword] = useState("")
  const [removeTarget, setRemoveTarget] = useState<MemberItem | null>(null)
  const [changePositionTarget, setChangePositionTarget] = useState<MemberItem | null>(null)
  const [orgSheetTarget, setOrgSheetTarget] = useState<MemberItem | null>(null)

  const canCreate = usePermission("org:assignment:create")
  const canUpdate = usePermission("org:assignment:update")
  const canDelete = usePermission("org:assignment:delete")

  const treeInitRef = useRef(false)

  const { data: treeData, isLoading: treeLoading } = useQuery({
    queryKey: ["departments", "tree"],
    queryFn: async () => {
      const res = await api.get<{ items: TreeNode[] }>("/api/v1/org/departments/tree")
      if (!treeInitRef.current && res.items.length > 0) {
        treeInitRef.current = true
        setExpanded(new Set(collectExpandedIds(res.items, 2)))
      }
      return res.items
    },
  })

  const effectiveDeptId = selectedDeptId

  const extraParams = useMemo(() => {
    return effectiveDeptId ? { departmentId: String(effectiveDeptId) } : undefined
  }, [effectiveDeptId])

  const {
    keyword,
    setKeyword,
    page,
    setPage,
    items,
    total,
    totalPages,
    isLoading,
    handleSearch,
  } = useListPage<MemberItem>({
    queryKey: "org-assignments",
    endpoint: "/api/v1/org/users",
    extraParams,
    enabled: !!effectiveDeptId,
  })

  const { data: positionsData } = useQuery({
    queryKey: ["positions", "all"],
    queryFn: async () => {
      const res = await api.get<{ items: PositionItem[] }>("/api/v1/org/positions?pageSize=9999")
      return res.items
    },
  })

  const { data: userSearchData } = useQuery({
    queryKey: ["users", "search", userKeyword],
    queryFn: async () => {
      const params = new URLSearchParams({ page: "1", pageSize: "50" })
      if (userKeyword) params.set("keyword", userKeyword)
      const res = await api.get<{ items: UserItem[] }>(`/api/v1/users?${params}`)
      return res.items
    },
    enabled: sheetOpen,
  })

  const positionMap = useMemo(() => {
    const map = new Map<number, string>()
    positionsData?.forEach((p) => map.set(p.id, p.name))
    return map
  }, [positionsData])

  const filteredTree = useMemo(() => {
    if (!treeData) return []
    return filterTree(treeData, deptSearch)
  }, [treeData, deptSearch])

  // Existing members' userIds for the current department (to gray out in add sheet)
  const existingUserIds = useMemo(() => {
    return new Set(items.map((m) => m.userId))
  }, [items])

  const selectedDept = useMemo(() => {
    if (!treeData || !effectiveDeptId) return null
    return findNodeById(treeData, effectiveDeptId)
  }, [treeData, effectiveDeptId])

  const allTreeIds = useMemo(() => collectAllIds(treeData ?? []), [treeData])

  const isAllExpanded = useMemo(() => {
    if (allTreeIds.length === 0) return false
    return allTreeIds.every((id) => expanded.has(id))
  }, [allTreeIds, expanded])

  const invalidateAll = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["org-assignments"] })
    queryClient.invalidateQueries({ queryKey: ["departments", "tree"] })
  }, [queryClient])

  const addMutation = useMutation({
    mutationFn: async () => {
      await api.post(`/api/v1/org/users/${selectedUserId}/positions`, {
        departmentId: effectiveDeptId,
        positionId: Number(selectedPositionId),
        isPrimary,
      })
    },
    onSuccess: () => {
      toast.success(t("org:assignments.addSuccess"))
      invalidateAll()
      setSheetOpen(false)
      setSelectedUserId("")
      setSelectedUserObj(null)
      setUserComboOpen(false)
      setSelectedPositionId("")
      setIsPrimary(false)
      setUserKeyword("")
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const removeMutation = useMutation({
    mutationFn: async (member: MemberItem) => {
      await api.delete(`/api/v1/org/users/${member.userId}/positions/${member.assignmentId}`)
    },
    onSuccess: () => {
      toast.success(t("org:assignments.removeSuccess"))
      invalidateAll()
      setRemoveTarget(null)
    },
    onError: (err: Error) => toast.error(err.message),
  })

  const setPrimaryMutation = useMutation({
    mutationFn: async (member: MemberItem) => {
      await api.put(`/api/v1/org/users/${member.userId}/positions/${member.assignmentId}/primary`, {})
    },
    onSuccess: () => {
      toast.success(t("org:assignments.primarySuccess"))
      queryClient.invalidateQueries({ queryKey: ["org-assignments"] })
    },
    onError: (err: Error) => toast.error(err.message),
  })

  function toggleExpand(id: number) {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <div className="flex min-h-[620px] flex-col gap-4 lg:h-[calc(100vh-104px)]">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">
          {t("org:assignments.title")}
        </h2>
      </div>

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-[304px_minmax(0,1fr)]">
        <section className="flex min-h-0 flex-col overflow-hidden rounded-xl border bg-card">
          <div className="border-b px-4 py-3">
            <div className="flex items-start justify-between gap-3">
              <h3 className="text-sm font-medium text-foreground">
                {t("org:departments.title")}
              </h3>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 px-2 text-xs text-muted-foreground hover:text-foreground"
                onClick={() => {
                  if (treeData && treeData.length > 0) {
                    const allIds = new Set(allTreeIds)
                    setExpanded(isAllExpanded ? new Set() : allIds)
                  }
                }}
              >
                {isAllExpanded ? t("common:collapseAll") : t("common:expandAll")}
              </Button>
            </div>
            <div className="relative mt-3">
              <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t("org:assignments.searchDept")}
                value={deptSearch}
                onChange={(e) => setDeptSearch(e.target.value)}
                className="h-9 pl-8"
              />
            </div>
          </div>

          {treeLoading ? (
            <div className="flex flex-1 items-center px-4 text-sm text-muted-foreground">
              {t("common:loading")}
            </div>
          ) : filteredTree.length === 0 ? (
            <div className="flex flex-1 items-center px-4 text-sm text-muted-foreground">
              {t("org:departments.empty")}
            </div>
          ) : (
            <div className="min-h-0 flex-1 overflow-auto px-2 py-3">
              {filteredTree.map((node) => (
                <DepartmentTreeItem
                  key={node.id}
                  node={node}
                  selectedId={effectiveDeptId}
                  onSelect={(id) => {
                    setSelectedDeptId(id)
                    setPage(1)
                  }}
                  expanded={expanded}
                  onToggleExpand={toggleExpand}
                  depth={0}
                />
              ))}
            </div>
          )}
        </section>

        <section className="flex min-h-0 flex-col overflow-hidden rounded-xl border bg-card">
          {!effectiveDeptId ? (
            <>
              <div className="border-b px-6 py-4">
                <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
                  <div>
                    <h3 className="truncate text-base font-semibold text-foreground">
                      {t("org:assignments.title")}
                    </h3>
                  </div>
                  {canCreate && (
                    <Button disabled>
                      <Plus className="mr-1.5 h-4 w-4" />
                      {t("org:assignments.addMember")}
                    </Button>
                  )}
                </div>
              </div>
              <div className="flex flex-1 items-center justify-center px-6 py-10">
                <div className="max-w-sm text-center">
                  <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                    <FolderOpen className="h-6 w-6" />
                  </div>
                  <p className="mt-5 text-base font-semibold text-foreground">
                    {t("org:assignments.selectDept")}
                  </p>
                  <p className="mt-2 text-sm leading-6 text-muted-foreground">
                    {t("org:assignments.selectDeptHint")}
                  </p>
                </div>
              </div>
            </>
          ) : (
            <>
              <div className="border-b px-6 py-4">
                <div className="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2.5">
                      <h3 className="truncate text-base font-semibold text-foreground">
                        {selectedDept?.name ?? "-"}
                      </h3>
                      {selectedDept?.code ? (
                        <Badge variant="outline" className="text-[10px] font-medium uppercase tracking-[0.14em] text-muted-foreground">
                          {selectedDept.code}
                        </Badge>
                      ) : null}
                    </div>
                    <p className="mt-1 text-sm text-muted-foreground">
                      {total} {t("org:assignments.memberCount")}
                    </p>
                  </div>

                  <div className="flex w-full flex-col gap-2 lg:w-auto lg:flex-row lg:items-center lg:justify-end">
                    <form onSubmit={handleSearch} className="flex w-full gap-2 lg:w-auto">
                      <div className="relative w-full lg:w-[280px]">
                        <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                        <Input
                          placeholder={t("org:assignments.searchPlaceholder")}
                          value={keyword}
                          onChange={(e) => setKeyword(e.target.value)}
                          className="h-9 pl-8"
                        />
                      </div>
                      <Button type="submit" variant="outline" className="h-9 shrink-0">
                        {t("common:search")}
                      </Button>
                    </form>
                    {canCreate && (
                      <Button onClick={() => setSheetOpen(true)} className="h-9 shrink-0">
                        <Plus className="mr-1.5 h-4 w-4" />
                        {t("org:assignments.addMember")}
                      </Button>
                    )}
                  </div>
                </div>
              </div>

              <div className="min-h-0 flex-1 overflow-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="sticky top-0 z-10 min-w-[220px] bg-card">
                        {t("org:assignments.user")}
                      </TableHead>
                      <TableHead className="sticky top-0 z-10 min-w-[140px] bg-card">
                        {t("org:assignments.position")}
                      </TableHead>
                      <TableHead className="sticky top-0 z-10 w-[96px] bg-card">
                        {t("org:assignments.type")}
                      </TableHead>
                      <TableHead className="sticky top-0 z-10 w-[140px] bg-card">
                        {t("org:assignments.assignedAt")}
                      </TableHead>
                      <DataTableActionsHead className="sticky top-0 z-10 bg-card" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {isLoading ? (
                      <DataTableLoadingRow colSpan={5} />
                    ) : items.length === 0 ? (
                      <DataTableEmptyRow
                        colSpan={5}
                        icon={Users}
                        title={t("org:assignments.empty")}
                        description={canCreate ? t("org:assignments.emptyHint") : undefined}
                      />
                    ) : (
                      items.map((item) => (
                        <TableRow key={item.assignmentId}>
                          <TableCell className="py-3.5">
                            <div className="flex items-center gap-3">
                              {item.avatar ? (
                                <img
                                  src={item.avatar}
                                  alt={item.username}
                                  className="h-9 w-9 shrink-0 rounded-full object-cover"
                                />
                              ) : (
                                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-semibold text-foreground/80">
                                  {item.username.charAt(0).toUpperCase()}
                                </div>
                              )}
                              <div className="min-w-0">
                                <p className="truncate text-sm font-medium text-foreground">
                                  {item.username}
                                </p>
                                {item.email && (
                                  <p className="truncate text-xs text-muted-foreground">
                                    {item.email}
                                  </p>
                                )}
                              </div>
                            </div>
                          </TableCell>
                          <TableCell className="text-sm text-foreground/90">
                            {positionMap.get(item.positionId) ?? "-"}
                          </TableCell>
                          <TableCell>
                            {item.isPrimary ? (
                              <Badge variant="default" className="px-2 text-[10px] font-medium">
                                {t("org:assignments.primary")}
                              </Badge>
                            ) : (
                              <Badge variant="outline" className="px-2 text-[10px] font-medium">
                                {t("org:assignments.secondary")}
                              </Badge>
                            )}
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground tabular-nums">
                            {item.createdAt ? new Date(item.createdAt).toLocaleDateString() : "-"}
                          </TableCell>
                          <DataTableActionsCell>
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button variant="ghost" size="icon-sm" className="rounded-lg">
                                  <MoreHorizontal className="h-4 w-4" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                {canUpdate && !item.isPrimary && (
                                  <DropdownMenuItem onClick={() => setPrimaryMutation.mutate(item)}>
                                    <Star className="mr-2 h-4 w-4" />
                                    {t("org:assignments.setPrimary")}
                                  </DropdownMenuItem>
                                )}
                                {canUpdate && (
                                  <DropdownMenuItem onClick={() => setChangePositionTarget(item)}>
                                    <ArrowRightLeft className="mr-2 h-4 w-4" />
                                    {t("org:assignments.changePosition")}
                                  </DropdownMenuItem>
                                )}
                                <DropdownMenuItem onClick={() => setOrgSheetTarget(item)}>
                                  <Building2 className="mr-2 h-4 w-4" />
                                  {t("org:assignments.viewOrgInfo")}
                                </DropdownMenuItem>
                                {canDelete && (
                                  <>
                                    <DropdownMenuSeparator />
                                    <DropdownMenuItem
                                      className="text-destructive focus:text-destructive"
                                      onClick={() => setRemoveTarget(item)}
                                    >
                                      <Trash2 className="mr-2 h-4 w-4" />
                                      {t("org:assignments.removeMember")}
                                    </DropdownMenuItem>
                                  </>
                                )}
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </DataTableActionsCell>
                        </TableRow>
                      ))
                    )}
                  </TableBody>
                </Table>
              </div>

              <div className="border-t border-border/60 px-6 py-4">
                <DataTablePagination
                  total={total}
                  page={page}
                  totalPages={totalPages}
                  onPageChange={setPage}
                  className="pt-0"
                />
              </div>
            </>
          )}
        </section>
      </div>

      {/* Add Member Sheet */}
      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent className="gap-0 p-0 sm:max-w-md">
          <SheetHeader className="border-b px-6 py-5">
            <SheetTitle>{t("org:assignments.addMemberTo", { dept: selectedDept?.name ?? "" })}</SheetTitle>
            <SheetDescription className="sr-only">
              {t("org:assignments.addMember")}
            </SheetDescription>
          </SheetHeader>
          <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
            <div className="flex-1 space-y-5 overflow-auto px-6 py-6">
              <div className="space-y-2">
              <label className="text-sm font-medium">{t("org:assignments.selectUser")}</label>
              <Popover open={userComboOpen} onOpenChange={setUserComboOpen}>
                <PopoverTrigger asChild>
                  <Button
                    variant="outline"
                    role="combobox"
                    aria-expanded={userComboOpen}
                    className="w-full justify-between font-normal"
                  >
                    {selectedUserObj ? (
                      <span className="flex items-center gap-2">
                        {selectedUserObj.avatar ? (
                          <img src={selectedUserObj.avatar} alt={selectedUserObj.username} className="h-5 w-5 rounded-full" />
                        ) : (
                          <div className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-[10px]">
                            {selectedUserObj.username.charAt(0).toUpperCase()}
                          </div>
                        )}
                        <span>{selectedUserObj.username}</span>
                        {selectedUserObj.email && <span className="text-xs text-muted-foreground">{selectedUserObj.email}</span>}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">{t("org:assignments.selectUser")}</span>
                    )}
                    <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start">
                  <div className="border-b p-2">
                    <Input
                      placeholder={t("org:assignments.searchUserPlaceholder")}
                      value={userKeyword}
                      onChange={(e) => setUserKeyword(e.target.value)}
                      className="h-8 border-0 p-0 shadow-none focus-visible:ring-0"
                    />
                  </div>
                  <div className="max-h-56 overflow-auto p-1">
                    {(!userSearchData || userSearchData.length === 0) && (
                      <p className="py-4 text-center text-sm text-muted-foreground">{t("common:noData")}</p>
                    )}
                    {userSearchData?.map((user) => {
                      const alreadyAssigned = existingUserIds.has(user.id)
                      return (
                        <button
                          key={user.id}
                          type="button"
                          disabled={alreadyAssigned}
                          onClick={() => {
                            setSelectedUserId(String(user.id))
                            setSelectedUserObj(user)
                            setUserComboOpen(false)
                          }}
                          className={cn(
                            "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none",
                            alreadyAssigned ? "cursor-not-allowed opacity-50" : "cursor-pointer hover:bg-accent",
                            String(user.id) === selectedUserId && "bg-accent"
                          )}
                        >
                          {user.avatar ? (
                            <img src={user.avatar} alt={user.username} className="h-5 w-5 rounded-full" />
                          ) : (
                            <div className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-[10px]">
                              {user.username.charAt(0).toUpperCase()}
                            </div>
                          )}
                          <span>{user.username}</span>
                          {user.email && <span className="text-xs text-muted-foreground">{user.email}</span>}
                          {alreadyAssigned && <span className="text-xs text-muted-foreground">({t("org:assignments.alreadyAssigned")})</span>}
                          {String(user.id) === selectedUserId && <Check className="ml-auto h-4 w-4" />}
                        </button>
                      )
                    })}
                  </div>
                </PopoverContent>
              </Popover>
            </div>

              <div className="space-y-2">
                <label className="text-sm font-medium">{t("org:assignments.selectPosition")}</label>
                <Select value={selectedPositionId} onValueChange={setSelectedPositionId}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder={t("org:assignments.selectPosition")} />
                  </SelectTrigger>
                  <SelectContent>
                    {positionsData?.filter((p) => p.isActive).map((pos) => (
                      <SelectItem key={pos.id} value={String(pos.id)}>
                        {pos.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center gap-2">
                <Checkbox
                  id="isPrimary"
                  checked={isPrimary}
                  onCheckedChange={(v) => setIsPrimary(v === true)}
                />
                <label htmlFor="isPrimary" className="cursor-pointer text-sm font-medium">
                  {t("org:assignments.setPrimary")}
                </label>
              </div>
            </div>

            <SheetFooter className="px-6 py-4">
              <Button variant="outline" onClick={() => setSheetOpen(false)}>
                {t("common:cancel")}
              </Button>
              <Button
                onClick={() => addMutation.mutate()}
                disabled={!selectedUserId || !selectedPositionId || addMutation.isPending}
              >
                {addMutation.isPending ? t("common:saving") : t("common:confirm")}
              </Button>
            </SheetFooter>
          </div>
        </SheetContent>
      </Sheet>

      {/* Remove Confirmation */}
      <AlertDialog open={!!removeTarget} onOpenChange={(open) => { if (!open) setRemoveTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("org:assignments.removeTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("org:assignments.removeDesc", { name: removeTarget?.username })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common:cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => removeTarget && removeMutation.mutate(removeTarget)}
              className="bg-destructive text-white hover:bg-destructive/90"
            >
              {t("org:assignments.confirmRemove")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Change Position Sheet */}
      {changePositionTarget && (
        <ChangePositionSheet
          open={!!changePositionTarget}
          onOpenChange={(open) => { if (!open) setChangePositionTarget(null) }}
          userId={changePositionTarget.userId}
          assignmentId={changePositionTarget.assignmentId}
          currentPositionId={changePositionTarget.positionId}
          onSuccess={invalidateAll}
        />
      )}

      {/* User Org Info Sheet */}
      <UserOrgSheet
        open={!!orgSheetTarget}
        onOpenChange={(open) => { if (!open) setOrgSheetTarget(null) }}
        userId={orgSheetTarget?.userId ?? null}
        username={orgSheetTarget?.username ?? ""}
        email={orgSheetTarget?.email ?? ""}
      />
    </div>
  )
}

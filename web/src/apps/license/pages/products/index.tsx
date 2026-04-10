import { useState } from "react"
import { useNavigate } from "react-router"
import { Plus, Search, Package, Eye, Pencil } from "lucide-react"
import { usePermission } from "@/hooks/use-permission"
import { useListPage } from "@/hooks/use-list-page"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  DataTableActions,
  DataTableActionsCell,
  DataTableActionsHead,
  DataTableCard,
  DataTableEmptyRow,
  DataTableLoadingRow,
  DataTablePagination,
  DataTableToolbar,
  DataTableToolbarGroup,
} from "@/components/ui/data-table"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { formatDateTime } from "@/lib/utils"
import { ProductSheet, type ProductItem } from "../../components/product-sheet"

const STATUS_MAP: Record<string, { label: string; variant: "default" | "secondary" | "outline" }> = {
  unpublished: { label: "未发布", variant: "secondary" },
  published: { label: "已发布", variant: "default" },
  archived: { label: "已归档", variant: "outline" },
}

export function Component() {
  const navigate = useNavigate()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<ProductItem | null>(null)
  const [statusFilter, setStatusFilter] = useState("")

  const canCreate = usePermission("license:product:create")
  const canUpdate = usePermission("license:product:update")

  const {
    keyword, setKeyword, page, setPage,
    items: products, total, totalPages, isLoading, handleSearch,
  } = useListPage<ProductItem>({
    queryKey: "license-products",
    endpoint: "/api/v1/license/products",
    extraParams: statusFilter ? { status: statusFilter } : undefined,
  })

  function handleCreate() {
    setEditing(null)
    setFormOpen(true)
  }

  function handleEdit(item: ProductItem) {
    setEditing(item)
    setFormOpen(true)
  }

  function handleStatusFilter(value: string) {
    setStatusFilter(value === "all" ? "" : value)
    setPage(1)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">商品管理</h2>
        {canCreate && (
          <Button size="sm" onClick={handleCreate}>
            <Plus className="mr-1.5 h-4 w-4" />
            新建商品
          </Button>
        )}
      </div>

      <DataTableToolbar>
        <DataTableToolbarGroup>
          <form onSubmit={handleSearch} className="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <div className="relative w-full sm:max-w-sm">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索名称或编码"
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                className="pl-8"
              />
            </div>
            <Select value={statusFilter || "all"} onValueChange={handleStatusFilter}>
              <SelectTrigger className="w-full sm:w-[130px]">
                <SelectValue placeholder="全部状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部状态</SelectItem>
                <SelectItem value="unpublished">未发布</SelectItem>
                <SelectItem value="published">已发布</SelectItem>
                <SelectItem value="archived">已归档</SelectItem>
              </SelectContent>
            </Select>
            <Button type="submit" variant="outline" size="sm">
              搜索
            </Button>
          </form>
        </DataTableToolbarGroup>
      </DataTableToolbar>

      <DataTableCard>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="min-w-[180px]">名称</TableHead>
              <TableHead className="w-[150px]">编码</TableHead>
              <TableHead className="w-[100px]">状态</TableHead>
              <TableHead className="w-[80px]">套餐数</TableHead>
              <TableHead className="w-[150px]">创建时间</TableHead>
              <DataTableActionsHead className="min-w-[140px]">操作</DataTableActionsHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <DataTableLoadingRow colSpan={6} />
            ) : products.length === 0 ? (
              <DataTableEmptyRow
                colSpan={6}
                icon={Package}
                title="暂无商品"
                description={canCreate ? "点击「新建商品」创建第一个商品" : undefined}
              />
            ) : (
              products.map((item) => {
                const status = STATUS_MAP[item.status] ?? { label: item.status, variant: "secondary" as const }
                return (
                  <TableRow key={item.id} className="cursor-pointer" onClick={() => navigate(`/license/products/${item.id}`)}>
                    <TableCell className="font-medium">{item.name}</TableCell>
                    <TableCell className="font-mono text-sm text-muted-foreground">{item.code}</TableCell>
                    <TableCell>
                      <Badge variant={status.variant}>{status.label}</Badge>
                    </TableCell>
                    <TableCell className="text-sm">{item.planCount}</TableCell>
                    <TableCell className="text-sm text-muted-foreground whitespace-nowrap">
                      {formatDateTime(item.createdAt)}
                    </TableCell>
                    <DataTableActionsCell>
                      <DataTableActions>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="px-2.5"
                          onClick={(e) => { e.stopPropagation(); navigate(`/license/products/${item.id}`) }}
                        >
                          <Eye className="mr-1 h-3.5 w-3.5" />
                          详情
                        </Button>
                        {canUpdate && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="px-2.5"
                            onClick={(e) => { e.stopPropagation(); handleEdit(item) }}
                          >
                            <Pencil className="mr-1 h-3.5 w-3.5" />
                            编辑
                          </Button>
                        )}
                      </DataTableActions>
                    </DataTableActionsCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </DataTableCard>

      <DataTablePagination
        total={total}
        page={page}
        totalPages={totalPages}
        onPageChange={setPage}
      />

      <ProductSheet open={formOpen} onOpenChange={setFormOpen} product={editing} />
    </div>
  )
}

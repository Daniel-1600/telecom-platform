import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Radio, Download, RefreshCw } from "lucide-react";

const profiles = [
  { id: "prof_001", imsi: "208930000000001", euiccId: "89049032004008882600001", status: "active", subscriber: "Alice Martin", activatedAt: "2026-03-15" },
  { id: "prof_002", imsi: "208930000000002", euiccId: "89049032004008882600002", status: "active", subscriber: "Bob Dupont", activatedAt: "2026-03-20" },
  { id: "prof_003", imsi: "208930000000003", euiccId: "89049032004008882600003", status: "downloading", subscriber: "Claire Moreau", activatedAt: "-" },
  { id: "prof_004", imsi: "208930000000005", euiccId: "89049032004008882600005", status: "active", subscriber: "Emma Petit", activatedAt: "2026-02-10" },
  { id: "prof_005", imsi: "208930000000008", euiccId: "89049032004008882600008", status: "inactive", subscriber: "Henri Faure", activatedAt: "-" },
];

const statusVariant = (s: string) =>
  s === "active" ? "success" : s === "downloading" ? "secondary" : s === "failed" ? "destructive" : "warning";

export default function ESIMPage() {
  return (
    <div className="p-8 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Radio className="size-8 text-primary" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight">eSIM Profiles</h1>
            <p className="text-muted-foreground mt-1">Manage eSIM profile provisioning via SM-DP+ (ES2+).</p>
          </div>
        </div>
        <Button size="sm"><Download className="size-4 mr-1.5" />Provision New</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>All Profiles</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-muted-foreground">
                  <th className="pb-3 font-medium">Profile ID</th>
                  <th className="pb-3 font-medium">Subscriber</th>
                  <th className="pb-3 font-medium">IMSI</th>
                  <th className="pb-3 font-medium">eUICC ID</th>
                  <th className="pb-3 font-medium">Status</th>
                  <th className="pb-3 font-medium text-right">Activated</th>
                  <th className="pb-3 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {profiles.map((p) => (
                  <tr key={p.id} className="border-b last:border-0 hover:bg-muted/50 transition-colors">
                    <td className="py-3 font-mono text-xs">{p.id}</td>
                    <td className="py-3">{p.subscriber}</td>
                    <td className="py-3 font-mono text-xs">{p.imsi}</td>
                    <td className="py-3 font-mono text-xs truncate max-w-[180px]">{p.euiccId}</td>
                    <td className="py-3"><Badge variant={statusVariant(p.status)}>{p.status}</Badge></td>
                    <td className="py-3 text-right text-muted-foreground text-xs">{p.activatedAt}</td>
                    <td className="py-3 text-right">
                      <Button variant="ghost" size="xs"><RefreshCw className="size-3" /></Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

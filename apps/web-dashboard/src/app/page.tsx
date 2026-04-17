import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatCard } from "@/components/stat-card";
import { Users, Activity, AlertTriangle, Server, Wifi, Database } from "lucide-react";

export default function DashboardPage() {
  return (
    <div className="p-8 space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="text-muted-foreground mt-1">Telecom platform overview and real-time metrics.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Subscribers"
          value="2,847"
          icon={Users}
          trend={{ value: 12.5, positive: true }}
          description="from last month"
        />
        <StatCard
          title="Active Sessions"
          value="342"
          icon={Activity}
          trend={{ value: 3.2, positive: true }}
          description="from last hour"
        />
        <StatCard
          title="Low Balance Alerts"
          value="18"
          icon={AlertTriangle}
          trend={{ value: -8, positive: true }}
          description="from yesterday"
        />
        <StatCard
          title="System Uptime"
          value="99.97%"
          icon={Server}
          description="last 30 days"
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Recent Subscribers</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { name: "Alice Martin", msisdn: "+33612345678", status: "active" as const },
                { name: "Bob Dupont", msisdn: "+33698765432", status: "active" as const },
                { name: "Claire Moreau", msisdn: "+33678901234", status: "provisioning" as const },
                { name: "David Leroy", msisdn: "+33667890123", status: "suspended" as const },
                { name: "Emma Petit", msisdn: "+33656789012", status: "active" as const },
              ].map((sub) => (
                <div key={sub.msisdn} className="flex items-center justify-between">
                  <div>
                    <p className="font-medium text-sm">{sub.name}</p>
                    <p className="text-xs text-muted-foreground">{sub.msisdn}</p>
                  </div>
                  <Badge
                    variant={
                      sub.status === "active" ? "success"
                        : sub.status === "suspended" ? "warning"
                        : "secondary"
                    }
                  >
                    {sub.status}
                  </Badge>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>System Status</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[
                { label: "API Server", icon: Server, status: "Healthy", ok: true },
                { label: "Database", icon: Database, status: "Connected", ok: true },
                { label: "Redis Cache", icon: Database, status: "Connected", ok: true },
                { label: "Charging Engine", icon: Activity, status: "Running", ok: true },
                { label: "AMF Gateway", icon: Wifi, status: "Connected", ok: true },
              ].map((svc) => (
                <div key={svc.label} className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <svc.icon className="size-4 text-muted-foreground" />
                    <span className="text-sm font-medium">{svc.label}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className={`size-2 rounded-full ${svc.ok ? "bg-emerald-500" : "bg-red-500"}`} />
                    <span className="text-xs text-muted-foreground">{svc.status}</span>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

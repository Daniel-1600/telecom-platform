import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export default function Home() {
  return (
    <div className="flex flex-col flex-1 items-center justify-center bg-zinc-50 font-sans dark:bg-black p-8">
      <main className="flex flex-1 w-full max-w-3xl flex-col items-center justify-center gap-8">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-center">Telecom Dashboard</CardTitle>
            <CardDescription className="text-center">
              Sovereign Telecom-as-a-Service Platform
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            <Button className="w-full">Get Started</Button>
            <Button variant="outline" className="w-full">View Documentation</Button>
          </CardContent>
        </Card>
        
        <div className="text-center text-sm text-zinc-600 dark:text-zinc-400">
          <p>Powered by Next.js and shadcn/ui</p>
        </div>
      </main>
    </div>
  );
}

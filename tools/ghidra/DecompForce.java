// Decompiles at the given entry addresses, creating the function first when the
// project has none there (read-only runs keep the change in memory only).
import ghidra.app.script.GhidraScript;
import ghidra.app.cmd.disassemble.DisassembleCommand;
import ghidra.app.decompiler.*;
import ghidra.program.model.address.*;
import ghidra.program.model.listing.*;
public class DecompForce extends GhidraScript {
  @Override public void run() throws Exception {
    DecompInterface d = new DecompInterface(); d.openProgram(currentProgram);
    for (String a : getScriptArgs()) {
      Address addr = toAddr(a);
      Function f = getFunctionAt(addr);
      if (f == null) {
        new DisassembleCommand(addr, null, true).applyTo(currentProgram, monitor);
        f = createFunction(addr, null);
      }
      println("==== " + (f == null ? "(none)" : f.getName()) + " @ " + addr);
      if (f == null) continue;
      DecompileResults r = d.decompileFunction(f, 120, monitor);
      println(r.decompileCompleted() ? r.getDecompiledFunction().getC() : "(failed)");
    }
  }
}

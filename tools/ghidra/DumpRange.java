// Prints the disassembly between two addresses: DumpRange.java START END
import ghidra.app.script.GhidraScript;
import ghidra.program.model.address.Address;
import ghidra.program.model.listing.Instruction;
public class DumpRange extends GhidraScript {
  @Override public void run() throws Exception {
    Address a = toAddr(getScriptArgs()[0]), end = toAddr(getScriptArgs()[1]);
    Instruction ins = getInstructionContaining(a);
    if (ins == null) { disassemble(a); ins = getInstructionContaining(a); }
    while (ins != null && ins.getAddress().compareTo(end) <= 0) {
      println("  " + ins.getAddress() + " " + ins);
      ins = ins.getNext();
    }
  }
}

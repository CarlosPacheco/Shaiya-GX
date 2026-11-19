// Lists functions whose entry lies in [START, END): ListRange.java START END
import ghidra.app.script.GhidraScript;
import ghidra.program.model.address.Address;
import ghidra.program.model.listing.*;
public class ListRange extends GhidraScript {
  @Override public void run() throws Exception {
    Address lo = toAddr(getScriptArgs()[0]), hi = toAddr(getScriptArgs()[1]);
    FunctionIterator it = currentProgram.getFunctionManager().getFunctions(lo, true);
    while (it.hasNext()) {
      Function f = it.next();
      if (f.getEntryPoint().compareTo(hi) >= 0) break;
      println("FN " + f.getEntryPoint() + " " + f.getBody().getNumAddresses() + " " + f.getName());
    }
  }
}

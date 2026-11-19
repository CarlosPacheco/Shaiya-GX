// Prints the address of every symbol whose name contains one of the arguments.
import ghidra.app.script.GhidraScript;
import ghidra.program.model.symbol.*;
public class SymAddr extends GhidraScript {
  @Override public void run() throws Exception {
    for (String w : getScriptArgs())
      for (Symbol s : currentProgram.getSymbolTable().getAllSymbols(true))
        if (s.getName(true).contains(w)) println("SYM " + s.getAddress() + " " + s.getName(true));
  }
}

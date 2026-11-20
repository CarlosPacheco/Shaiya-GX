// Lists every instruction with a scalar operand in [LO, HI] (decimal) and its function:
// ScalarRange.java LO HI
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.scalar.Scalar;
public class ScalarRange extends GhidraScript {
  @Override public void run() throws Exception {
    long lo = Long.parseLong(getScriptArgs()[0]), hi = Long.parseLong(getScriptArgs()[1]);
    for (Instruction ins : currentProgram.getListing().getInstructions(true)) {
      for (int i = 0; i < ins.getNumOperands(); i++) {
        for (Object o : ins.getOpObjects(i)) {
          if (o instanceof Scalar s && s.getUnsignedValue() >= lo && s.getUnsignedValue() <= hi && s.bitLength() >= 16) {
            Function f = getFunctionContaining(ins.getAddress());
            println(ins.getAddress() + " " + (f == null ? "?" : f.getName()) + " " + ins + "   [" + s.getUnsignedValue() + "]");
          }
        }
      }
    }
  }
}

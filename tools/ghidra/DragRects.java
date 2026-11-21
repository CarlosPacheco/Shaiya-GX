// For every function, prints the float constants stored into [reg+0xc]/[reg+0x10]
// through FLD+ftol (the frame size used by the common drag helper 0x526f30)
// plus the texture names that function references.
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.address.*;
import ghidra.program.model.data.*;
public class DragRects extends GhidraScript {
  @Override public void run() throws Exception {
    FunctionIterator it = currentProgram.getFunctionManager().getFunctions(true);
    while (it.hasNext()) {
      Function f = it.next();
      InstructionIterator ii = currentProgram.getListing().getInstructions(f.getBody(), true);
      Float last = null; boolean ftol = false; Float w = null, h = null; StringBuilder tex = new StringBuilder();
      while (ii.hasNext()) {
        Instruction in = ii.next(); String m = in.getMnemonicString();
        if (m.equals("FLD") && in.getNumOperands()==1) {
          Object[] o = in.getOpObjects(0);
          if (o.length>0 && o[0] instanceof ghidra.program.model.scalar.Scalar) {
            try { int b = currentProgram.getMemory().getInt(toAddr(((ghidra.program.model.scalar.Scalar)o[0]).getUnsignedValue())); last = Float.intBitsToFloat(b); } catch (Exception e) { last=null; }
          } else if (o.length>0 && o[0] instanceof Address) {
            try { last = Float.intBitsToFloat(currentProgram.getMemory().getInt((Address)o[0])); } catch (Exception e) { last=null; }
          }
        } else if (m.equals("CALL") && in.getDefaultOperandRepresentation(0).toLowerCase().contains("5cd414")) { ftol = last!=null; }
        else if (m.equals("MOV") && ftol && in.getNumOperands()==2) {
          String s = in.getDefaultOperandRepresentation(0);
          if (s.matches(".*\\+ 0xc\\]")) w = last; else if (s.matches(".*\\+ 0x10\\]")) h = last;
          ftol = false;
        }
        for (int k=0;k<in.getNumOperands();k++) for (Object o : in.getOpObjects(k)) if (o instanceof ghidra.program.model.scalar.Scalar) {
          long v = ((ghidra.program.model.scalar.Scalar)o).getUnsignedValue();
          if (v>=0x6b0000 && v<0x6d0000) { Data d = getDataAt(toAddr(v)); if (d!=null && d.getValue() instanceof String) { String s=(String)d.getValue(); if (s.toLowerCase().endsWith(".tga")&&!tex.toString().contains(s)) tex.append(s).append(' '); } }
        }
      }
      if (w!=null || h!=null) println(f.getEntryPoint()+" w="+w+" h="+h+" "+tex);
    }
  }
}
